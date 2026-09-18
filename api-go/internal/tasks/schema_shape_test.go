package tasks

import (
	"context"
	"os"
	"testing"

	"neuroboost/api-go/internal/database"
)

// Does the database actually have the shape recurring tasks need?
//
// 🔴 Asked of the DATABASE, not of the migration chain that built it. Every test
// database here is built by running the migrations from zero, and from zero they
// are correct by construction — which is exactly why the divergence that took
// production down on 18.08 was invisible to the whole suite (gotcha 18:
// CREATE TABLE IF NOT EXISTS accepted a foreign table and recorded success).
//
// ⚠ Skips without DATABASE_URL, like every DB-backed test in this repository.
// That is a real weakness and it is named here rather than left to be
// discovered: on 18.09 a skipping test let a broken ON CONFLICT ship, so where a
// contract CAN be checked without a database it should be
// (see reminders/conflict_target_test.go).
func TestRecurringTaskShape(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	ctx := context.Background()

	t.Run("task carries the repeat columns", func(t *testing.T) {
		present := columnsOf(t, d, ctx, "task")
		// The floor: no columns at all means the query found the wrong place,
		// and every check below would pass by finding nothing to complain about.
		if len(present) == 0 {
			t.Fatal("no columns found for `task` — this test proved nothing")
		}
		for _, c := range []string{"rrule", "repeat_anchor", "nag_minutes"} {
			if !present[c] {
				t.Errorf("task.%s is missing — recurring tasks cannot work", c)
			}
		}
		// 🔴 And the reverse: task.event_id must NOT come back. The link is
		// event.task_id and has been since the baseline; 000017 added a second
		// column for the same relationship by mistake and 000018 removed it.
		// Two columns for one link drift apart — one gets set, the other does
		// not, and two readers answer the same question differently.
		if present["event_id"] {
			t.Error("task.event_id is back; the link belongs on event.task_id (see 000018)")
		}
	})

	t.Run("event carries the nag interval", func(t *testing.T) {
		present := columnsOf(t, d, ctx, "event")
		if len(present) == 0 {
			t.Fatal("no columns found for `event` — this test proved nothing")
		}
		if !present["nag_minutes"] {
			t.Error("event.nag_minutes is missing; Denis asked for the nag interval on events too")
		}
	})

	t.Run("reminder can count its nags", func(t *testing.T) {
		present := columnsOf(t, d, ctx, "reminder")
		if len(present) == 0 {
			t.Fatal("no columns found for `reminder` — this test proved nothing")
		}
		if !present["nag_count"] {
			t.Error("reminder.nag_count is missing; nagging would never stop")
		}
	})

	t.Run("occurrences are unique per day", func(t *testing.T) {
		present := columnsOf(t, d, ctx, "task_occurrence")
		if len(present) == 0 {
			t.Fatal("task_occurrence does not exist")
		}
		for _, c := range []string{"task_id", "occurrence", "state"} {
			if !present[c] {
				t.Errorf("task_occurrence.%s is missing", c)
			}
		}

		// 🔴 Without the unique key a double tap writes two «done» rows, and
		// every count of "how often did I actually do this" is wrong for ever.
		var n int
		err := d.Pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_indexes
			 WHERE tablename = 'task_occurrence'
			   AND indexdef ILIKE '%UNIQUE%'
			   AND indexdef ILIKE '%task_id%'
			   AND indexdef ILIKE '%occurrence%'`).Scan(&n)
		if err != nil {
			t.Fatalf("read indexes: %v", err)
		}
		if n == 0 {
			t.Error("task_occurrence has no unique (task_id, occurrence) index")
		}
	})
}

func columnsOf(t *testing.T, d *database.DB, ctx context.Context, table string) map[string]bool {
	t.Helper()
	rows, err := d.Pool.Query(ctx,
		`SELECT column_name FROM information_schema.columns
		  WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		t.Fatalf("read columns of %s: %v", table, err)
	}
	defer rows.Close()

	present := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan: %v", err)
		}
		present[c] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return present
}
