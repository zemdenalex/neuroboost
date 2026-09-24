package daytasks

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

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/tasks"
)

const ny = "America/New_York"

// dayDB is a user in New York with an empty day-tasks history.
func dayDB(t *testing.T) (*database.DB, context.Context, string) {
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
	tasks.InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()
	var id string
	email := fmt.Sprintf("daytasks-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, timezone) VALUES ($1, $2) RETURNING id`, email, ny).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id) })
	return d, ctx, id
}

func asUser(req *http.Request, userID string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
}

// newTask goes through the tasks API's own door, not an INSERT.
func newTask(t *testing.T, userID string, body map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	tasks.CreateHandler(rec, asUser(httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(b)), userID))
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create task: %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Data.ID == "" {
		t.Fatalf("create task: no id in %s", rec.Body.String())
	}
	return got.Data.ID
}

// closeTask is «✅ Готово» through PATCH, which is what sets completed_at.
func closeTask(t *testing.T, userID, taskID string) {
	t.Helper()
	req := asUser(httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID,
		bytes.NewReader([]byte(`{"status":"DONE"}`))), userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	tasks.UpdateHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("close task: %d %s", rec.Code, rec.Body.String())
	}
}

// 🔴 gotcha 18: ask the database for the shape, do not trust the chain that built it.
func TestDayCommitmentTablesHaveTheirShape(t *testing.T) {
	d, ctx, _ := dayDB(t)
	for table, cols := range map[string][]string{
		"day_commitment":     {"user_id", "day", "task_id", "added_at", "removed_at"},
		"day_commitment_day": {"user_id", "day", "confirmed_at"},
	} {
		for _, c := range cols {
			var n int
			if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
				WHERE table_name = $1 AND column_name = $2`, table, c).Scan(&n); err != nil || n != 1 {
				t.Errorf("%s.%s: found %d (%v) — migration 000022 not applied?", table, c, n, err)
			}
		}
	}
}

// Spec §1, §2, §3 in one day: 5 promised, 3 closed today, one closed tomorrow
// does not count, the day is confirmed → 3 of 5, 🟧.
func TestADayCountsWhatWasClosedOnIt(t *testing.T) {
	_, ctx, user := dayDB(t)
	now := time.Now()
	today := Today(now, ny)

	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, newTask(t, user, map[string]any{"title": fmt.Sprintf("дело %d", i), "priority": 2}))
	}
	if err := Confirm(ctx, user, today, ids); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	for _, id := range ids[:3] {
		closeTask(t, user, id)
	}

	days, err := List(ctx, user, today, today)
	if err != nil || len(days) != 1 {
		t.Fatalf("list: %v %+v", err, days)
	}
	got := days[0]
	if got.Done != 3 || got.Target != 5 || got.Level != 3 || !got.Confirmed || len(got.Items) != 5 {
		t.Errorf("day = %+v, want done 3 of 5, level 3 (🟧), confirmed, 5 items", got)
	}
}

// Denis 22.09: a day never taken is ⬛ however much was done on it.
func TestAnUnconfirmedDayIsBlackEvenIfDone(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "без подтверждения"})
	if err := Add(ctx, user, today, id); err != nil {
		t.Fatalf("add: %v", err)
	}
	closeTask(t, user, id)
	days, _ := List(ctx, user, today, today)
	if days[0].Level != 0 || days[0].Confirmed {
		t.Errorf("day = %+v, want level 0 and not confirmed", days[0])
	}
}

// Spec §2: a done task cannot be added — «add the finished one, get green».
func TestADoneTaskCannotBeAdded(t *testing.T) {
	_, ctx, user := dayDB(t)
	id := newTask(t, user, map[string]any{"title": "уже сделано"})
	closeTask(t, user, id)
	if err := Add(ctx, user, Today(time.Now(), ny), id); err != ErrNotOpen {
		t.Errorf("add done task: err = %v, want ErrNotOpen", err)
	}
}

// Spec §2: a series counts its own day, marked through the series door.
func TestASeriesDayCountsWhenThatDayIsDone(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "таблетки", "rrule": "FREQ=DAILY",
		"due_date": today.Format(time.RFC3339)})
	if err := Confirm(ctx, user, today, []string{id}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := tasks.MarkOccurrence(ctx, user, id, today, tasks.StateDone); err != nil {
		t.Fatalf("mark: %v", err)
	}
	days, _ := List(ctx, user, today, today)
	if days[0].Done != 1 {
		t.Errorf("series day not counted: %+v", days[0])
	}
	// The same series is open again tomorrow: adding it there is allowed.
	if err := Add(ctx, user, today.AddDate(0, 0, 1), id); err != nil {
		t.Errorf("tomorrow's day of the series refused: %v", err)
	}
}

// Spec §5: yesterday cannot be edited; tomorrow can.
func TestRemovingFollowsTheNoonRule(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "убрать"})
	seedPromise(t, d, user, today.AddDate(0, 0, -1), id)
	if err := Add(ctx, user, today.AddDate(0, 0, 1), id); err != nil {
		t.Fatalf("add tomorrow: %v", err)
	}
	if err := Remove(ctx, user, today.AddDate(0, 0, -1), id, time.Now()); err != ErrTooLate {
		t.Errorf("remove from yesterday: err = %v, want ErrTooLate", err)
	}
	if err := Remove(ctx, user, today.AddDate(0, 0, 1), id, time.Now()); err != nil {
		t.Errorf("remove from tomorrow: %v", err)
	}
	days, _ := List(ctx, user, today.AddDate(0, 0, 1), today.AddDate(0, 0, 1))
	if len(days[0].Items) != 0 {
		t.Errorf("removed task still listed: %+v", days[0].Items)
	}
}

// Every day of the range is returned, empty ones as [] (gotcha 6).
func TestListReturnsEveryDayWithEmptyArrays(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	days, err := List(ctx, user, today, today.AddDate(0, 0, 2))
	if err != nil || len(days) != 3 {
		t.Fatalf("list: %v, %d days", err, len(days))
	}
	b, _ := json.Marshal(days[1])
	if !bytes.Contains(b, []byte(`"items":[]`)) {
		t.Errorf("empty day serialises as %s, want items []", b)
	}
}

// A stranger's task cannot enter my day.
func TestAnotherUsersTaskIsNotFound(t *testing.T) {
	_, ctx, user := dayDB(t)
	_, _, other := dayDB(t)
	theirs := newTask(t, other, map[string]any{"title": "чужое"})
	if err := Add(ctx, user, Today(time.Now(), ny), theirs); err != ErrTaskNotFound {
		t.Errorf("add stranger's task: err = %v, want ErrTaskNotFound", err)
	}
}

// «Closed on that day» is the USER's date, and the check must be able to fail
// at any hour. New York alone cannot do that: its date equals UTC's for 20
// hours a day, so dropping «AT TIME ZONE» would pass most of the time. The zone
// is chosen so that its date differs from UTC's right now: UTC−12 before noon
// UTC, UTC+14 after.
func TestADayIsTheUsersDateNotUTCs(t *testing.T) {
	d, ctx, user := dayDB(t)
	zone := "Etc/GMT-14" // UTC+14 (POSIX sign is inverted)
	if time.Now().UTC().Hour() < 12 {
		zone = "Etc/GMT+12" // UTC−12
	}
	if _, err := d.Pool.Exec(ctx, `UPDATE "user" SET timezone = $2 WHERE id = $1`, user, zone); err != nil {
		t.Fatalf("set zone: %v", err)
	}
	today := Today(time.Now(), zone)
	if today.Format(dateFmt) == time.Now().UTC().Format(dateFmt) {
		t.Fatalf("zone %s has UTC's date right now; the test would prove nothing", zone)
	}
	id := newTask(t, user, map[string]any{"title": "по местной дате"})
	if err := Confirm(ctx, user, today, []string{id}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	closeTask(t, user, id)
	days, err := List(ctx, user, today, today)
	if err != nil || days[0].Done != 1 {
		t.Errorf("closed today in %s, counted %+v (%v); want done 1", zone, days, err)
	}
}

// Spec §4: already-promised first, then yesterday's undone, then due, then
// priority; never a done task, never more than N.
func TestProposalOrder(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1)

	pinned := newTask(t, user, map[string]any{"title": "заранее", "priority": 5})
	carried := newTask(t, user, map[string]any{"title": "вчерашнее", "priority": 5})
	due := newTask(t, user, map[string]any{"title": "срок сегодня", "priority": 4,
		"due_date": today.Format(time.RFC3339)})
	urgent := newTask(t, user, map[string]any{"title": "срочно", "priority": 1})
	buffer := newTask(t, user, map[string]any{"title": "буфер", "priority": 0})
	done := newTask(t, user, map[string]any{"title": "сделано", "priority": 1})
	extra := newTask(t, user, map[string]any{"title": "лишнее", "priority": 3})
	closeTask(t, user, done)

	if err := Add(ctx, user, today, pinned); err != nil {
		t.Fatal(err)
	}
	seedPromise(t, d, user, yesterday, carried)

	got, err := Propose(ctx, user, today)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	var ids []string
	for _, it := range got {
		ids = append(ids, it.TaskID)
	}
	want := []string{pinned, carried, due, urgent, extra} // N=5: buffer is sixth, done never
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Errorf("proposal = %v\nwant       %v (pinned, carried, due, urgent, extra)", ids, want)
	}
	_ = buffer

	// Proposing writes nothing.
	days, _ := List(ctx, user, today, today)
	if days[0].Confirmed || len(days[0].Items) != 1 {
		t.Errorf("proposal wrote something: %+v", days[0])
	}
}

func callDay(t *testing.T, h http.HandlerFunc, userID, method, target string, body any, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := asUser(httptest.NewRequest(method, target, rd), userID)
	if params != nil {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// The bot's whole path through HTTP: propose → confirm → close → read the colour.
func TestTheDayThroughHTTP(t *testing.T) {
	_, _, user := dayDB(t)
	today := Today(time.Now(), ny).Format("2006-01-02")
	id := newTask(t, user, map[string]any{"title": "одно дело", "priority": 1})

	rec := callDay(t, ProposalHandler, user, http.MethodGet, "/api/day-tasks/proposal?day="+today, nil, nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(id)) {
		t.Fatalf("proposal: %d %s", rec.Code, rec.Body.String())
	}
	rec = callDay(t, ConfirmHandler, user, http.MethodPost, "/api/day-tasks/confirm",
		map[string]any{"day": today, "task_ids": []string{id}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	closeTask(t, user, id)
	rec = callDay(t, ListHandler, user, http.MethodGet, "/api/day-tasks?from="+today+"&to="+today, nil, nil)
	var got struct {
		Data []Day `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || len(got.Data) != 1 || got.Data[0].Done != 1 || got.Data[0].Level != 1 {
		t.Errorf("list: %d %+v — want 1 of 5 done, level 1 (🟫)", rec.Code, got.Data)
	}
}

func TestHTTPErrorsHaveTheirCodes(t *testing.T) {
	d, _, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	id := newTask(t, user, map[string]any{"title": "x"})
	seedPromise(t, d, user, today.AddDate(0, 0, -1), id)
	done := newTask(t, user, map[string]any{"title": "done"})
	closeTask(t, user, done)

	cases := []struct {
		name string
		rec  *httptest.ResponseRecorder
		code int
		err  string
	}{
		{"remove from yesterday", callDay(t, RemoveHandler, user, http.MethodDelete, "/", nil,
			map[string]string{"day": yesterday, "task_id": id}), 409, "TOO_LATE"},
		{"add done", callDay(t, AddHandler, user, http.MethodPost, "/",
			map[string]any{"day": today.Format("2006-01-02"), "task_id": done}, nil), 409, "NOT_OPEN"},
		{"bad day", callDay(t, AddHandler, user, http.MethodPost, "/",
			map[string]any{"day": "22.09.2026", "task_id": id}, nil), 400, "INVALID_DAY"},
		{"huge range", callDay(t, ListHandler, user, http.MethodGet,
			"/?from=2026-01-01&to=2026-12-31", nil, nil), 400, "RANGE_TOO_LARGE"},
		// Not a UUID: the cast fails inside Postgres, which must still read as
		// «no such task», not as a server error.
		{"bad task id", callDay(t, AddHandler, user, http.MethodPost, "/",
			map[string]any{"day": today.Format("2006-01-02"), "task_id": "abc"}, nil), 404, "TASK_NOT_FOUND"},
		{"no user", callDay(t, ListHandler, "", http.MethodGet, "/?from=2026-01-01&to=2026-01-02", nil, nil), 401, "NOT_AUTHENTICATED"},
	}
	for _, c := range cases {
		if c.rec.Code != c.code || !bytes.Contains(c.rec.Body.Bytes(), []byte(c.err)) {
			t.Errorf("%s: %d %s — want %d %s", c.name, c.rec.Code, c.rec.Body.String(), c.code, c.err)
		}
	}
}

// A promised task whose calendar the user can no longer see (left a shared
// calendar, the task moved) leaves the day: its title is no longer theirs to
// read. The row stays — access may come back — but List does not show it.
func TestListShowsOnlyTasksStillVisible(t *testing.T) {
	d, ctx, user := dayDB(t)
	_, _, other := dayDB(t)
	today := Today(time.Now(), ny)
	theirs := newTask(t, other, map[string]any{"title": "больше не моё"})
	// The row as it would be after access was lost: written directly, because
	// Add rightly refuses a task the user cannot see.
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO day_commitment (user_id, day, task_id) VALUES ($1, $2, $3)`,
		user, today.Format(dateFmt), theirs); err != nil {
		t.Fatalf("seed: %v", err)
	}
	days, err := List(ctx, user, today, today)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(days[0].Items) != 0 {
		t.Errorf("a task from a calendar the user cannot see is listed: %+v", days[0].Items)
	}
}

// Spec §11: the start is the first day the person took; days before it (or
// every day, for someone who never took one) come back before_start.
func TestBeforeStartIsTheFirstTakenDay(t *testing.T) {
	_, _, user := dayDB(t)
	today := Today(time.Now(), ny)
	iso, yest := today.Format("2006-01-02"), today.AddDate(0, 0, -1).Format("2006-01-02")
	list := func() []Day {
		rec := callDay(t, ListHandler, user, http.MethodGet, "/api/day-tasks?from="+yest+"&to="+iso, nil, nil)
		if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"before_start"`)) {
			t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
		}
		var got struct {
			Data []Day `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		return got.Data
	}
	if d := list(); !d[0].BeforeStart || !d[1].BeforeStart {
		t.Errorf("never took a day: want both before_start, got %+v", d)
	}
	id := newTask(t, user, map[string]any{"title": "дело", "priority": 1})
	if rec := callDay(t, ConfirmHandler, user, http.MethodPost, "/api/day-tasks/confirm",
		map[string]any{"day": iso, "task_ids": []string{id}}, nil); rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	if d := list(); !d[0].BeforeStart || d[1].BeforeStart {
		t.Errorf("took today: want yesterday before, today not; got %+v", d)
	}
}
