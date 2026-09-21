package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/events"
	"neuroboost/api-go/internal/middleware"
)

// Задача ↔ событие через дверь клиента, против настоящей базы.
//
// ⚠ Нужны DATABASE_URL и -count=1: без первого тест скипается и печатает ok,
// без второго Go отдаёт кэш — чужая база не входит в то, что он хэширует.

func callTaskRoute(h http.HandlerFunc, path, taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	rec := httptest.NewRecorder()
	h(rec, req.WithContext(ctx))
	return rec
}

func callConvert(taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	return callTaskRoute(ConvertHandler, "/api/tasks/"+taskID+"/convert", taskID, userID, body)
}

func setZone(t *testing.T, userID, tz string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		`UPDATE "user" SET timezone = $2 WHERE id = $1`, userID, tz); err != nil {
		t.Fatalf("set zone: %v", err)
	}
}

type storedEvent struct {
	ID, Timezone    string
	Rrule           *string
	TaskID          *string
	StartsAt, Ends  time.Time
	Description     *string
	Tags            []string
	ReminderOffsets []int
}

func loadEvent(t *testing.T, id string) storedEvent {
	t.Helper()
	var e storedEvent
	if err := db.Pool.QueryRow(context.Background(), `
		SELECT id::text, COALESCE(timezone,''), rrule, task_id::text, starts_at, ends_at,
		       description, COALESCE(tags,'{}'), COALESCE(reminder_offsets,'{}')
		  FROM event WHERE id = $1`, id).Scan(&e.ID, &e.Timezone, &e.Rrule, &e.TaskID,
		&e.StartsAt, &e.Ends, &e.Description, &e.Tags, &e.ReminderOffsets); err != nil {
		t.Fatalf("load event %s: %v", id, err)
	}
	return e
}

func createdEventID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var got struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Data.ID == "" {
		t.Fatalf("no event id in %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	return got.Data.ID
}

func TestConvertStoresTheUsersZone(t *testing.T) {
	d, ctx, user := repeatDB(t)
	setZone(t, user, "America/New_York")

	task, err := insertTask(ctx, user, CreateTaskRequest{Title: "созвон"}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "starts_at": "2026-10-20T09:00:00-04:00", "ends_at": "2026-10-20T10:00:00-04:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, ev.ID) })

	if ev.Timezone != "America/New_York" {
		t.Errorf("event.timezone = %q, want the user's America/New_York", ev.Timezone)
	}
}

// 🔴 Поведенческая половина: что именно ломает чужая зона. Серия в 09:00 по
// Нью-Йорку, развёрнутая после 01.11.2026 (США переходят на зимнее время),
// обязана остаться в 09:00. С «Europe/Moscow» (без перехода) она съезжает на 08:00.
//
// ⚠ Колонку проверяет тест выше; этот нужен, потому что момент starts_at верен
// и с чужой зоной — RFC3339 несёт смещение. Тест на «событие в его час» был бы
// зелёным до починки.
func TestAScheduledSeriesKeepsItsLocalHourAcrossDST(t *testing.T) {
	d, ctx, user := repeatDB(t)
	setZone(t, user, "America/New_York")

	due := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	dueStr := due.Format("2006-01-02")
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "зарядка", Rrule: str("FREQ=DAILY"), DueDate: &dueStr,
	}, &due)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "repeat": "series",
		"starts_at": "2026-10-28T09:00:00-04:00", "ends_at": "2026-10-28T09:30:00-04:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))

	ny, _ := time.LoadLocation("America/New_York")
	from := time.Date(2026, 11, 3, 0, 0, 0, 0, ny)
	occ := events.OccurrencesInRange(events.Event{
		StartsAt: ev.StartsAt, EndsAt: ev.Ends, Rrule: ev.Rrule, Timezone: ev.Timezone,
	}, from, from.AddDate(0, 0, 1), nil)
	if len(occ) != 1 {
		t.Fatalf("expanded %d occurrences on 03.11, want 1 (rrule %v)", len(occ), ev.Rrule)
	}
	if h := occ[0].In(ny).Hour(); h != 9 {
		t.Errorf("03.11 occurrence at %02d:00 New York, want 09:00 — the series drifted with a foreign zone", h)
	}
}
