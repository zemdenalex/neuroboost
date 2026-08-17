package tasks

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
)

// setupTestDB connects to DATABASE_URL and seeds a user + task, returning the
// task ID, user ID, and a cleanup func. Skips the test if DATABASE_URL is unset.
// Seeds only baseline-guaranteed columns (id/email on "user", user_id/title on
// task) and relies on column defaults for everything else, so it survives
// regardless of which later migrations have run.
func setupTestDB(t *testing.T) (taskID, userID string, cleanup func()) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	InitDB(d)
	// Task access now goes through calendar membership (CalendarIDsFor), which
	// reads the calendars package's own db pointer — set it here too, or any
	// handler under test panics on a nil pool.
	calendars.InitDB(d)
	ctx := context.Background()

	// Unique email avoids UNIQUE collisions if a prior run's cleanup was skipped.
	email := fmt.Sprintf("pomodoro-test-%d@example.com", time.Now().UnixNano())
	err = d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`,
		email,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// The task's own personal calendar — calendar_id is NOT NULL (migration
	// 000012), and access to the task now comes from calendar membership,
	// not user_id, so the fixture needs a real calendar row to join.
	var calID string
	err = d.Pool.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Мой календарь', 'personal') RETURNING id`,
		userID,
	).Scan(&calID)
	if err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	_, err = d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, 'owner', 'active')`,
		calID, userID,
	)
	if err != nil {
		t.Fatalf("seed calendar_member: %v", err)
	}

	err = d.Pool.QueryRow(ctx,
		`INSERT INTO task (user_id, calendar_id, title) VALUES ($1, $2, $3) RETURNING id`,
		userID, calID, "Test task",
	).Scan(&taskID)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	cleanup = func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE user_id = $1`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
		d.Close()
	}
	return taskID, userID, cleanup
}

// callLogTime issues a log-time request authenticated as userID.
func callLogTime(taskID, userID string, minutes int) *httptest.ResponseRecorder {
	body, _ := json.Marshal(LogTimeRequest{Minutes: minutes})
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/log-time", bytes.NewReader(body))
	// Inject chi URL param + authenticated user into context.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	LogTimeHandler(rec, req)
	return rec
}

func TestLogTimeHandler_AddsAndClampsMinutes(t *testing.T) {
	taskID, userID, cleanup := setupTestDB(t)
	defer cleanup()

	// Add 25 minutes.
	rec := callLogTime(taskID, userID, 25)
	if rec.Code != http.StatusOK {
		t.Fatalf("add: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data Task `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ActualMinutes != 25 {
		t.Fatalf("expected 25 actual_minutes, got %d", resp.Data.ActualMinutes)
	}

	// Undo (-25) should bring it to 0.
	rec = callLogTime(taskID, userID, -25)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ActualMinutes != 0 {
		t.Fatalf("expected 0 after undo, got %d", resp.Data.ActualMinutes)
	}

	// Over-subtract clamps at 0, never negative.
	rec = callLogTime(taskID, userID, -100)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ActualMinutes != 0 {
		t.Fatalf("expected clamp at 0, got %d", resp.Data.ActualMinutes)
	}
}

func TestLogTimeHandler_NotOwnedReturns404(t *testing.T) {
	taskID, _, cleanup := setupTestDB(t)
	defer cleanup()

	rec := callLogTime(taskID, "00000000-0000-0000-0000-000000000000", 25)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %d", rec.Code)
	}
}
