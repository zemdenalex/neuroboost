package export

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

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

// setupTestDB connects to DATABASE_URL and seeds a user, returning the user ID
// and a cleanup func. Skips the test if DATABASE_URL is unset (matches the
// tasks package pattern, so these run in CI but are skipped on bare local runs).
func setupTestDB(t *testing.T) (userID string, cleanup func()) {
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
	// queryTasks now scopes access via calendar membership (CalendarIDsFor),
	// not user_id directly — that lookup dereferences the calendars package's
	// own db pointer, which only this call sets.
	calendars.InitDB(d)
	ctx := context.Background()

	email := fmt.Sprintf("export-test-%d@example.com", time.Now().UnixNano())
	err = d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`,
		email,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	cleanup = func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE user_id = $1`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
		d.Close()
	}
	return userID, cleanup
}

// TestExportIncludesActualMinutes guards the export side: a task with logged
// Pomodoro time must carry actual_minutes into the export payload. Goes RED
// when queryTasks omits actual_minutes from its SELECT (field defaults to 0).
func TestExportIncludesActualMinutes(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// The task's own personal calendar — calendar_id is NOT NULL (migration
	// 000012), and access to the task now comes from calendar membership,
	// not user_id, so the fixture needs a real calendar row to join.
	var calID string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Мой календарь', 'personal') RETURNING id`,
		userID,
	).Scan(&calID)
	if err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, 'owner', 'active')`,
		calID, userID,
	)
	if err != nil {
		t.Fatalf("seed calendar_member: %v", err)
	}

	var taskID string
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO task (user_id, calendar_id, title, actual_minutes) VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, calID, "Logged task", 25,
	).Scan(&taskID)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	tasks, err := queryTasks(ctx, userID)
	if err != nil {
		t.Fatalf("queryTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ActualMinutes != 25 {
		t.Fatalf("export dropped actual_minutes: expected 25, got %d", tasks[0].ActualMinutes)
	}
}

// TestImportPreservesActualMinutes guards the import side. It imports a task
// with a FRESH id (one not already in the DB) so ImportHandler's
// "ON CONFLICT (id) DO NOTHING" does NOT short-circuit — this mirrors the real
// restore-into-clean-DB path. Goes RED when the INSERT omits actual_minutes
// (the column then falls back to its DEFAULT 0).
func TestImportPreservesActualMinutes(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	freshID := "11111111-1111-1111-1111-111111111111"
	now := time.Now().UTC()
	body, _ := json.Marshal(ImportRequest{
		Version: "0.4.8",
		Tasks: []TaskRow{{
			ID:            freshID,
			Title:         "Restored task",
			Status:        "TODO",
			Priority:      3,
			ActualMinutes: 25,
			Tags:          []string{},
			Contexts:      []string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		}},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/import", bytes.NewReader(body))
	ctxReq := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctxReq)
	rec := httptest.NewRecorder()
	ImportHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var stored int
	err := db.Pool.QueryRow(ctx,
		`SELECT actual_minutes FROM task WHERE id = $1 AND user_id = $2`,
		freshID, userID,
	).Scan(&stored)
	if err != nil {
		t.Fatalf("read back task: %v", err)
	}
	if stored != 25 {
		t.Fatalf("import dropped actual_minutes: expected 25, got %d", stored)
	}
}
