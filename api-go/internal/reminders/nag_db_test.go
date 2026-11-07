package reminders

import (
	"context"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
)

// «Постоянные напоминания» (Denis, 07.11): an unanswered reminder comes back
// until it is answered or switched off — not six times, not only until
// midnight. What still stops it: an answer, a closed task, and a newer
// reminder of the same item (only the newest one nags).
func TestNagGoesOnUntilAnsweredButOnlyTheNewestAndNotForClosedTasks(t *testing.T) {
	cleanup := setupScanDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := seedTelegramUser(t, "nag", time.Now().UnixNano()%1_000_000_000+11)
	if _, err := db.Pool.Exec(ctx,
		`UPDATE "user" SET settings = '{"reminders":{"quiet_hours_respected":false}}' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	cal, err := calendars.Create(ctx, userID, "Постоянные", nil)
	if err != nil {
		t.Fatal(err)
	}
	task := func(title, status string) string {
		var id string
		if err := db.Pool.QueryRow(ctx,
			`INSERT INTO task (user_id, calendar_id, title, status, nag_minutes) VALUES ($1, $2, $3, $4, 10) RETURNING id`,
			userID, cal.ID, title, status).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, id) })
		return id
	}
	now := time.Now().UTC().Truncate(time.Second)
	// Yesterday's occurrence and seven nags already: both old limits would
	// have stopped this one.
	row := func(taskID string, minutesBefore int, remindAt time.Time) string {
		var id string
		if err := db.Pool.QueryRow(ctx, `
			INSERT INTO reminder (user_id, source_kind, task_id, occurrence_start, minutes_before,
			                      remind_at, status, channel, message, nag_count)
			VALUES ($1, 'TASK', $2, $3, $4, $5, 'SENT', 'TELEGRAM', 'x', 7) RETURNING id`,
			userID, taskID, now.Add(-24*time.Hour), minutesBefore, remindAt).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	open := task("отчёт", "TODO")
	older := row(open, 120, now.Add(-30*time.Minute))
	newest := row(open, 60, now.Add(-time.Minute))
	closed := row(task("уже сделано", "DONE"), 60, now.Add(-time.Minute))

	if _, err := NagUnanswered(ctx, now, nil); err != nil {
		t.Fatalf("nag: %v", err)
	}
	status := func(id string) (string, time.Time) {
		var st string
		var at time.Time
		if err := db.Pool.QueryRow(ctx, `SELECT status, remind_at FROM reminder WHERE id = $1`, id).Scan(&st, &at); err != nil {
			t.Fatal(err)
		}
		return st, at
	}
	if st, at := status(newest); st != "PENDING" || !at.Equal(now.Add(10*time.Minute)) {
		t.Errorf("the newest unanswered reminder: %s at %v, want PENDING at +10 min — the old cap or midnight stop is back", st, at)
	}
	if st, _ := status(older); st != "SENT" {
		t.Errorf("an older reminder of the same task nagged too (%s): every row would multiply the nag", st)
	}
	// Retired for good, not merely skipped this tick: once the newest row is
	// re-armed into the future it no longer LOOKS newest by remind_at, and a
	// skipped row would nag on the next tick (review 07.11).
	var retired bool
	if err := db.Pool.QueryRow(ctx, `SELECT answered_at IS NOT NULL FROM reminder WHERE id = $1`, older).Scan(&retired); err != nil {
		t.Fatal(err)
	}
	if !retired {
		t.Error("the older reminder was skipped, not retired: it nags again as soon as the newest is re-armed")
	}

	// A row that waited five hours comes back ONCE, ten minutes from now —
	// not every missed nag replayed a minute apart.
	late := row(task("ждал", "TODO"), 60, now.Add(-5*time.Hour))
	if _, err := NagUnanswered(ctx, now, nil); err != nil {
		t.Fatal(err)
	}
	if _, at := status(late); !at.Equal(now.Add(10 * time.Minute)) {
		t.Errorf("a five-hour-old reminder re-armed for %v, want now + 10 min", at)
	}
	if st, _ := status(closed); st != "SENT" {
		t.Errorf("a closed task's reminder nagged (%s)", st)
	}
}
