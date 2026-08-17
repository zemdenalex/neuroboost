package events

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// An occurrence may be excepted once per SERIES, not once per person.
//
// 🔴 Until migration 000013 the constraint was
// UNIQUE (user_id, event_id, occurrence), so two writing members of one shared
// calendar could each detach the same occurrence: the second insert did not
// conflict. The calendar then showed TWO replacement events while the original
// occurrence was hidden once — fetchExceptions reads exceptions by calendar and
// ignores user_id on purpose, so it hid the occurrence a single time no matter
// how many rows existed. Nothing resolved that state on its own.
//
// Two writing members is not a slice-3 hypothetical: TransferOwnership demotes
// the previous owner to editor and leaves the new owner beside them.
//
// Skips without DATABASE_URL, which CI sets. See CLAUDE.md for the throwaway
// Postgres recipe — running this without a database proves nothing, and this
// repository has been bitten by exactly that twice.
func TestOneExceptionPerOccurrenceAcrossMembers(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()

	ownerID := seedUser(t, ctx, d, "owner")
	editorID := seedUser(t, ctx, d, "editor")

	cal, err := calendars.Create(ctx, ownerID, "Shared", nil)
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, $4)`,
		cal.ID, editorID, calendars.RoleEditor, calendars.StatusActive); err != nil {
		t.Fatalf("seed editor membership: %v", err)
	}

	// A weekly series owned by the calendar's owner.
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	var seriesID string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, all_day, rrule, timezone)
		 VALUES ($1, $2, 'Weekly', $3, $4, false, 'FREQ=WEEKLY', 'UTC') RETURNING id`,
		ownerID, cal.ID, start, start.Add(time.Hour)).Scan(&seriesID); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	occurrence := start.AddDate(0, 0, 7)
	parent := Event{
		Title: "Weekly", StartsAt: occurrence, EndsAt: occurrence.Add(time.Hour),
		Timezone: "UTC", IsWorkEvent: true,
		// NOT NULL in the schema; nil would fail the insert before the test
		// reached what it is actually about.
		ReminderOffsets: []int{}, Tags: []string{},
	}

	// Both members detach the SAME occurrence, one after the other.
	if _, err := detachOccurrence(ctx, ownerID, parent, seriesID, occurrence); err != nil {
		t.Fatalf("owner detach: %v", err)
	}
	if _, err := detachOccurrence(ctx, editorID, parent, seriesID, occurrence); err != nil {
		t.Fatalf("editor detach: %v", err)
	}

	var exceptions int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM event_exception WHERE event_id = $1 AND occurrence = $2`,
		seriesID, occurrence).Scan(&exceptions); err != nil {
		t.Fatalf("count exceptions: %v", err)
	}
	if exceptions != 1 {
		t.Errorf("want 1 exception for the occurrence, got %d — the second member's detach created its own row", exceptions)
	}

	// The visible symptom, asserted separately: the row count could be right
	// while orphaned replacements still render on the calendar.
	var replacements int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM event WHERE calendar_id = $1 AND rrule IS NULL AND task_id IS NULL`,
		cal.ID).Scan(&replacements); err != nil {
		t.Fatalf("count replacements: %v", err)
	}
	if replacements != 1 {
		t.Errorf("want 1 replacement event on the calendar, got %d — the extra copies are what members actually see", replacements)
	}
}

func seedUser(t *testing.T, ctx context.Context, d *database.DB, label string) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("exc-%s-%d@example.com", label, time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("seed %s: %v", label, err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id)
	})
	return id
}
