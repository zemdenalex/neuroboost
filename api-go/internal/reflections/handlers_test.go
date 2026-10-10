package reflections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

// The first tests of this package (docs/agents/queue.md «Постоянная работа»).
// Everything goes through the HTTP handlers against a real database: a test
// that writes rows directly cannot notice a handler that never writes them.
//
// ⚠ Needs DATABASE_URL and -count=1. Without the first it skips and still
// prints ok; without the second Go serves a cached result.

func testDB(t *testing.T) *database.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	InitDB(d)
	return d
}

func seedUser(t *testing.T, d *database.DB, label string) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("reflections-%s-%d@example.com", label, time.Now().UnixNano())
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, id) })
	return id
}

func seedEvent(t *testing.T, d *database.DB, user string) string {
	t.Helper()
	ctx := context.Background()
	var cal, ev string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Личный', 'personal') RETURNING id`,
		user).Scan(&cal); err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	start := time.Now().UTC().Add(-2 * time.Hour)
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at)
		 VALUES ($1, $2, 'созвон', $3, $4) RETURNING id`,
		user, cal, start, start.Add(time.Hour)).Scan(&ev); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	return ev
}

type call struct {
	method, path string
	params       map[string]string
	body         any
	user         string
}

func (c call) do(h http.HandlerFunc) *httptest.ResponseRecorder {
	var raw []byte
	if c.body != nil {
		raw, _ = json.Marshal(c.body)
	}
	req := httptest.NewRequest(c.method, c.path, bytes.NewReader(raw))
	rctx := chi.NewRouteContext()
	for k, v := range c.params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, c.user)
	rec := httptest.NewRecorder()
	h(rec, req.WithContext(ctx))
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) Reflection {
	t.Helper()
	var got struct {
		Data Reflection `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return got.Data
}

func postForEvent(user, event string, body map[string]any) call {
	return call{method: http.MethodPost, path: "/api/events/" + event + "/reflection",
		params: map[string]string{"id": event}, body: body, user: user}
}

// 🔴 The defect this file found. The event editor saves a reflection with
// every event save, and until 25.09 it sent was_completed/was_on_time = true
// hardcoded; the upsert overwrote them unconditionally. «Не успел вовремя»,
// set on the Reflections page, was reset by renaming the event. A repeat
// POST that does not name a flag must leave it as it was.
func TestARepeatSaveKeepsTheFlagsItDoesNotName(t *testing.T) {
	d := testDB(t)
	user := seedUser(t, d, "flags")
	event := seedEvent(t, d, user)

	first := postForEvent(user, event, map[string]any{"mood": 4, "was_completed": false, "was_on_time": false}).do(CreateForEventHandler)
	if first.Code != http.StatusCreated {
		t.Fatalf("first save: %d %s", first.Code, first.Body.String())
	}

	again := postForEvent(user, event, map[string]any{"mood": 7}).do(CreateForEventHandler)
	if again.Code != http.StatusCreated {
		t.Fatalf("second save: %d %s", again.Code, again.Body.String())
	}
	got := decode(t, again)
	if got.WasCompleted || got.WasOnTime {
		t.Errorf("flags after a save that did not name them: completed=%v on_time=%v, want false false",
			got.WasCompleted, got.WasOnTime)
	}
	if got.Mood == nil || *got.Mood != 7 {
		t.Errorf("mood %v, want 7: the named field still updates", got.Mood)
	}
}

// A NEW reflection that names no flags reads as «done, on time», as before.
func TestANewReflectionDefaultsToDoneOnTime(t *testing.T) {
	d := testDB(t)
	user := seedUser(t, d, "defaults")
	event := seedEvent(t, d, user)

	rec := postForEvent(user, event, map[string]any{"energy": 5}).do(CreateForEventHandler)
	if rec.Code != http.StatusCreated {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	got := decode(t, rec)
	if !got.WasCompleted || !got.WasOnTime {
		t.Errorf("new reflection: completed=%v on_time=%v, want true true", got.WasCompleted, got.WasOnTime)
	}
}

func TestAScaleOutsideOneToTenIsRefused(t *testing.T) {
	d := testDB(t)
	user := seedUser(t, d, "scale")
	event := seedEvent(t, d, user)

	for _, body := range []map[string]any{{"energy": 0}, {"mood": 11}, {"focus": -1}} {
		if rec := postForEvent(user, event, body).do(CreateForEventHandler); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: status %d, want 400", body, rec.Code)
		}
	}
	// Positive control: the edges themselves are accepted.
	if rec := postForEvent(user, event, map[string]any{"energy": 1, "mood": 10}).do(CreateForEventHandler); rec.Code != http.StatusCreated {
		t.Errorf("1 and 10: status %d, want 201: %s", rec.Code, rec.Body.String())
	}
}

// One user can neither read nor change another's reflection.
func TestAReflectionBelongsToItsAuthor(t *testing.T) {
	d := testDB(t)
	author := seedUser(t, d, "author")
	other := seedUser(t, d, "other")
	event := seedEvent(t, d, author)

	created := decode(t, postForEvent(author, event, map[string]any{"notes": "устал"}).do(CreateForEventHandler))
	if created.ID == "" {
		t.Fatal("no reflection created")
	}

	patch := call{method: http.MethodPatch, path: "/api/reflections/" + created.ID,
		params: map[string]string{"id": created.ID}, body: map[string]any{"notes": "чужое"}, user: other}.do(UpdateHandler)
	if patch.Code != http.StatusNotFound {
		t.Errorf("another user's PATCH: status %d, want 404", patch.Code)
	}

	list := call{method: http.MethodGet, path: "/api/reflections", user: other}.do(ListHandler)
	var got struct {
		Data []Reflection `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(got.Data) != 0 {
		t.Errorf("another user's list has %d reflections, want 0", len(got.Data))
	}

	// Positive control: the author's own PATCH goes through and changes the row.
	own := call{method: http.MethodPatch, path: "/api/reflections/" + created.ID,
		params: map[string]string{"id": created.ID}, body: map[string]any{"notes": "отдохнул"}, user: author}.do(UpdateHandler)
	if own.Code != http.StatusOK {
		t.Fatalf("author's PATCH: %d %s", own.Code, own.Body.String())
	}
	if n := decode(t, own).Notes; n == nil || *n != "отдохнул" {
		t.Errorf("notes after the author's PATCH: %v", n)
	}
}
