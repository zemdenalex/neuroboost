package reminders

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/remindrule"
)

// «Remind me regularly until the deadline» end to end on a real database:
// the rule goes in through its API door (PUT …/remind-rule — not an INSERT
// past it, the 20.09 lesson: a test that skips the door does not prove the
// door exists), and the scanner turns it into journal rows.
func TestARuleBecomesRemindersUpToTheDeadline(t *testing.T) {
	cleanup := setupScanDB(t)
	defer cleanup()
	ctx := context.Background()
	remindrule.InitDB(db)

	userID := seedTelegramUser(t, "rule", time.Now().UnixNano()%1_000_000_000+7)
	// No quiet hours: the moments below must not depend on the clock of the run.
	if _, err := db.Pool.Exec(ctx,
		`UPDATE "user" SET settings = '{"reminders":{"quiet_hours_respected":false}}' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	cal, err := calendars.Create(ctx, userID, "Напоминания до срока", nil)
	if err != nil {
		t.Fatalf("calendar: %v", err)
	}

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID)))
		})
	})
	router.Put("/api/tasks/{id}/remind-rule", remindrule.Put(remindrule.TaskKind))
	router.Put("/api/events/{id}/remind-rule", remindrule.Put(remindrule.EventKind))
	put := func(path, body string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, path, strings.NewReader(body)))
		return rec.Code
	}

	// 1. A task due on a DATE (stored as midnight): «каждый день в 12» must
	// remind at 12:00 ON the deadline day — the day a date-only deadline
	// would otherwise silently skip.
	day := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 3)
	var taskID string
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO task (user_id, calendar_id, title, status, due_date)
		 VALUES ($1, $2, 'экзамен', 'TODO', $3) RETURNING id`, userID, cal.ID, day).Scan(&taskID); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, taskID) })
	if code := put("/api/tasks/"+taskID+"/remind-rule", `{"every_days":1,"times":["12:00"]}`); code != http.StatusOK {
		t.Fatalf("PUT task rule: %d", code)
	}

	// 2. An event in an hour, every 5 minutes: an 11-minute window holds
	// exactly three moments, aligned to the start.
	start := time.Now().UTC().Truncate(time.Minute).Add(2 * time.Hour)
	var eventID string
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, timezone)
		 VALUES ($1, $2, 'созвон', $3, $4, 'UTC') RETURNING id`,
		userID, cal.ID, start, start.Add(time.Hour)).Scan(&eventID); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, eventID) })
	if code := put("/api/events/"+eventID+"/remind-rule", `{"every_minutes":5}`); code != http.StatusOK {
		t.Fatalf("PUT event rule: %d", code)
	}

	for _, w := range [][2]time.Time{
		{day.Add(11 * time.Hour), day.Add(13 * time.Hour)},
		{start.Add(-time.Hour), start.Add(-time.Hour + 11*time.Minute)},
	} {
		if _, err := Scan(ctx, w[0], w[1], slog.Default()); err != nil {
			t.Fatalf("scan: %v", err)
		}
	}

	var taskAt []time.Time
	rows, err := db.Pool.Query(ctx, `SELECT remind_at FROM reminder WHERE task_id = $1 ORDER BY remind_at`, taskID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var at time.Time
		_ = rows.Scan(&at)
		taskAt = append(taskAt, at)
	}
	rows.Close()
	if len(taskAt) != 1 || !taskAt[0].Equal(day.Add(12*time.Hour)) {
		t.Errorf("task rule rows = %v, want one at %v (12:00 on the deadline day)", taskAt, day.Add(12*time.Hour))
	}

	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM reminder WHERE event_id = $1`, eventID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("event rule rows = %d, want 3 (every 5 minutes over 11)", n)
	}

	// A task with no due has nothing to run up to: refused, not stored.
	var bare string
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO task (user_id, calendar_id, title, status) VALUES ($1, $2, 'без срока', 'TODO') RETURNING id`,
		userID, cal.ID).Scan(&bare); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, bare) })
	if code := put("/api/tasks/"+bare+"/remind-rule", `{"every_minutes":30}`); code != http.StatusBadRequest {
		t.Errorf("rule on a task with no due: %d, want 400", code)
	}
}
