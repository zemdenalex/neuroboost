package events

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

// Событие → задача через обработчик, против настоящей базы.
//
// ⚠ Нужны DATABASE_URL и -count=1: без первого тест скипается и печатает ok,
// без второго Go отдаёт кэш.

func toTaskDB(t *testing.T) (*database.DB, context.Context, string, string) {
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
	calendars.InitDB(d)
	ctx := context.Background()
	user := seedUser(t, ctx, d, "totask")
	calID, err := calendars.PersonalIDFor(ctx, user)
	if err != nil {
		t.Fatalf("personal calendar: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE user_id = $1`, user) })
	return d, ctx, user, calID
}

func callToTask(id, userID string, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+id+"/to-task", bytes.NewReader(raw))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	rec := httptest.NewRecorder()
	ToTaskHandler(rec, req.WithContext(ctx))
	return rec
}

type toTaskReply struct {
	Data struct {
		Task struct {
			ID               string  `json:"id"`
			DueDate          string  `json:"due_date"`
			EstimatedMinutes *int    `json:"estimated_minutes"`
			Rrule            *string `json:"rrule"`
		} `json:"task"`
		Lost    []string `json:"lost"`
		EventID *string  `json:"event_id"`
	} `json:"data"`
}

func decodeToTask(t *testing.T, rec *httptest.ResponseRecorder) toTaskReply {
	t.Helper()
	var r toTaskReply
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
	}
	return r
}

func seedSeries(t *testing.T, ctx context.Context, d *database.DB, user, cal, rule string, start time.Time) string {
	t.Helper()
	var id string
	if err := d.Pool.QueryRow(ctx, `
		INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, all_day, timezone, rrule)
		VALUES ($1, $2, 'планёрка', $3, $4, false, 'Europe/Moscow', $5) RETURNING id`,
		user, cal, start, start.Add(time.Hour), rule).Scan(&id); err != nil {
		t.Fatalf("seed series: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, id) })
	return id
}

func eventGone(ctx context.Context, d *database.DB, id string) bool {
	var n int
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM event WHERE id = $1`, id).Scan(&n)
	return n == 0
}

func TestMovingAPlainEventLeavesOnlyTheTask(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	msk, _ := time.LoadLocation("Europe/Moscow")
	start := time.Date(2026, 10, 20, 14, 50, 0, 0, msk)
	ev := seedEvent(t, ctx, d, user, cal, "врач", start) // one hour long

	rec := callToTask(ev, user, map[string]any{"mode": "move"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	r := decodeToTask(t, rec)
	if r.Data.Task.ID == "" || r.Data.Task.DueDate != "2026-10-20" {
		t.Errorf("task = %+v, want an id and due 2026-10-20", r.Data.Task)
	}
	if r.Data.Task.EstimatedMinutes == nil || *r.Data.Task.EstimatedMinutes != 60 {
		t.Errorf("estimate = %v, want 60", r.Data.Task.EstimatedMinutes)
	}
	if !eventGone(ctx, d, ev) {
		t.Error("move left the event in place")
	}
}

func TestLinkKeepsTheEventPointingAtTheNewTask(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedEvent(t, ctx, d, user, cal, "созвон", time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC))

	rec := callToTask(ev, user, map[string]any{"mode": "link"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	r := decodeToTask(t, rec)
	var link *string
	_ = d.Pool.QueryRow(ctx, `SELECT task_id::text FROM event WHERE id = $1`, ev).Scan(&link)
	if link == nil || *link != r.Data.Task.ID {
		t.Errorf("event.task_id = %v, want %s", link, r.Data.Task.ID)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedEvent(t, ctx, d, user, cal, "проба", time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC))

	rec := callToTask(ev, user, map[string]any{"mode": "move", "dry_run": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var n int
	_ = d.Pool.QueryRow(ctx, `SELECT count(*) FROM task WHERE user_id = $1`, user).Scan(&n)
	if n != 0 || eventGone(ctx, d, ev) {
		t.Errorf("dry run wrote: tasks=%d eventGone=%v", n, eventGone(ctx, d, ev))
	}
	if r := decodeToTask(t, rec); len(r.Data.Lost) == 0 || r.Data.Lost[0] != LostStartTime {
		t.Errorf("dry run must name what is lost, got %v", r.Data.Lost)
	}
}

func TestASeriesEventMustSayWhichPart(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedSeries(t, ctx, d, user, cal, "FREQ=DAILY", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC))
	rec := callToTask(ev, user, map[string]any{"mode": "link"})
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("REPEAT_CHOICE_REQUIRED")) {
		t.Errorf("got %d %s, want 400 REPEAT_CHOICE_REQUIRED", rec.Code, rec.Body.String())
	}
}

// 🔴 «Только этот раз» + перенести пропускает ОДНО вхождение. deleteEvent здесь
// снёс бы всю серию — и в тесте это видно, только если спросить про серию.
func TestOnceMoveSkipsOneOccurrenceAndKeepsTheSeries(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedSeries(t, ctx, d, user, cal, "FREQ=DAILY", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC))

	rec := callToTask(ev+":2026-10-22", user, map[string]any{"mode": "move", "repeat": "once"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	r := decodeToTask(t, rec)
	if r.Data.Task.DueDate != "2026-10-22" || r.Data.Task.Rrule != nil {
		t.Errorf("task = %+v, want a one-off due 2026-10-22", r.Data.Task)
	}
	if eventGone(ctx, d, ev) {
		t.Fatal("the whole series was deleted for one day")
	}
	var skipped int
	_ = d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM event_exception WHERE event_id = $1 AND skipped`, ev).Scan(&skipped)
	if skipped != 1 {
		t.Errorf("exceptions = %d, want 1 skipped day", skipped)
	}
}

func TestOnceLinkDetachesThatDayOntoTheTask(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedSeries(t, ctx, d, user, cal, "FREQ=DAILY", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC))

	rec := callToTask(ev, user, map[string]any{
		"mode": "link", "repeat": "once", "occurrence": "2026-10-23",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	r := decodeToTask(t, rec)
	if r.Data.EventID == nil || *r.Data.EventID == ev {
		t.Fatalf("event_id = %v, want the detached occurrence, not the series", r.Data.EventID)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, *r.Data.EventID) })
	var link *string
	_ = d.Pool.QueryRow(ctx, `SELECT task_id::text FROM event WHERE id = $1`, *r.Data.EventID).Scan(&link)
	if link == nil || *link != r.Data.Task.ID {
		t.Errorf("detached occurrence task_id = %v, want %s", link, r.Data.Task.ID)
	}
	var seriesLink *string
	_ = d.Pool.QueryRow(ctx, `SELECT task_id::text FROM event WHERE id = $1`, ev).Scan(&seriesLink)
	if seriesLink != nil {
		t.Errorf("the whole series got linked to a one-off task: %v", *seriesLink)
	}
}

func TestOnceWithoutADayIsRefused(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedSeries(t, ctx, d, user, cal, "FREQ=DAILY", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC))
	rec := callToTask(ev, user, map[string]any{"mode": "move", "repeat": "once"})
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("OCCURRENCE_REQUIRED")) {
		t.Errorf("got %d %s, want 400 OCCURRENCE_REQUIRED", rec.Code, rec.Body.String())
	}
}

func TestSeriesToTaskCarriesTheRule(t *testing.T) {
	d, ctx, user, cal := toTaskDB(t)
	ev := seedSeries(t, ctx, d, user, cal, "FREQ=WEEKLY", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC))
	rec := callToTask(ev, user, map[string]any{"mode": "move", "repeat": "series"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	r := decodeToTask(t, rec)
	if r.Data.Task.Rrule == nil || *r.Data.Task.Rrule != "FREQ=WEEKLY" {
		t.Errorf("rrule = %v, want FREQ=WEEKLY", r.Data.Task.Rrule)
	}
	var anchor *time.Time
	_ = d.Pool.QueryRow(ctx, `SELECT repeat_anchor FROM task WHERE id = $1`, r.Data.Task.ID).Scan(&anchor)
	if anchor == nil {
		t.Error("a rule without an anchor is a task that says it repeats and never does")
	}
	if !eventGone(ctx, d, ev) {
		t.Error("series move left the series in place")
	}
}
