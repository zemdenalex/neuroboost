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

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/middleware"
)

// Two refusals from the audit of 25.09
// (docs/team/research/V003-20260925-res-audit-today-and-scope.md, M3 and M4),
// through the HTTP handlers against a real database.
//
// ⚠ Needs DATABASE_URL and -count=1 (see callOccurrence).

// M4: postpone_days had no ceiling. One request of a million days was a
// million INSERTs and a series closed for 2700 years. The bot offers at most
// 365 (maxPostponeDays); the API refuses past a year, like ListOccurrences.
func TestPostponingMoreThanAYearIsRefused(t *testing.T) {
	d, ctx, user := repeatDB(t)
	task, err := insertTask(ctx, user, CreateTaskRequest{Title: "зарядка", Rrule: str("FREQ=DAILY")}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	if rec := callOccurrence(task.ID, user, map[string]any{"postpone_days": 400}); rec.Code != http.StatusBadRequest {
		t.Errorf("400 days: status %d, want 400", rec.Code)
	}
	if n := occurrenceRows(t, task.ID); n != 0 {
		t.Errorf("a refused postpone wrote %d rows", n)
	}
	// Positive control: the bot's own maximum still goes through.
	if rec := callOccurrence(task.ID, user, map[string]any{"postpone_days": 365}); rec.Code != http.StatusOK {
		t.Errorf("365 days: status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

// M3: a read-only member scheduling a shared task got «500 SCHEDULE_ERROR»,
// «the app is broken», instead of «read only».
func TestAViewerSchedulingASharedTaskIsToldItIsReadOnly(t *testing.T) {
	d, ctx, owner := repeatDB(t)
	viewer := seedTaskUser(t, ctx, d, "sched-viewer")

	shared, err := calendars.Create(ctx, owner, "Дом", nil)
	if err != nil {
		t.Fatalf("create shared calendar: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM calendar WHERE id = $1`, shared.ID) })
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, $4)`,
		shared.ID, viewer, calendars.RoleViewer, calendars.StatusActive); err != nil {
		t.Fatalf("seed viewer: %v", err)
	}
	task, err := insertTask(ctx, owner, CreateTaskRequest{Title: "купить хлеб", CalendarID: shared.ID}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
	body, _ := json.Marshal(ScheduleTaskRequest{
		StartsAt: start.Format(time.RFC3339), EndsAt: start.Add(time.Hour).Format(time.RFC3339),
	})
	call := func(user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+task.ID+"/schedule", bytes.NewReader(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", task.ID)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		c = context.WithValue(c, middleware.UserIDKey, user)
		rec := httptest.NewRecorder()
		ScheduleHandler(rec, req.WithContext(c))
		return rec
	}

	if rec := call(viewer); rec.Code != http.StatusForbidden {
		t.Errorf("viewer: status %d, want 403: %s", rec.Code, rec.Body.String())
	}
	// Positive control: the owner schedules the same task.
	if rec := call(owner); rec.Code != http.StatusCreated {
		t.Errorf("owner: status %d, want 201: %s", rec.Code, rec.Body.String())
	}
}
