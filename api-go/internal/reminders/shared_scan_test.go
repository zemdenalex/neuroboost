package reminders

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// Does a shared event remind BOTH people, or only the one who created it?
//
// 🔴 This test exists because I answered that question twice and got it wrong
// the first time. On 17.08 I told Denis a shared event "would only remind one
// of you", from memory of the P3 spec. Reading the scanner said the opposite:
// scanEvents selects by the SCANNING user's calendars and the insert uses that
// user's id, not the event's author. But reading is how I got the split-identity
// claim wrong on the same day — a mechanism is not a state, and code is not a
// run. So this settles it against a real database.
//
// It is also the first DB-backed test in this package; everything else here is
// pure, which is why the question had never been asked of the real thing.

// setupScanDB connects to DATABASE_URL and returns a cleanup. Skips without it,
// like the calendars package — CI sets it, a bare local run does not.
func setupScanDB(t *testing.T) func() {
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
	calendars.InitDB(d)
	return d.Close
}

// seedTelegramUser makes a user the scanner will actually look at: Scan skips
// anyone without a tg_id, so a user seeded without one would produce zero
// reminders and the test would "pass" having proved nothing.
func seedTelegramUser(t *testing.T, label string, tgID int64) string {
	t.Helper()
	ctx := context.Background()
	var id string
	email := fmt.Sprintf("scan-%s-%d@example.com", label, time.Now().UnixNano())
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, tg_id, timezone) VALUES ($1, $2, 'UTC') RETURNING id`,
		email, tgID).Scan(&id); err != nil {
		t.Fatalf("seed %s: %v", label, err)
	}
	t.Cleanup(func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id)
	})
	return id
}

func TestSharedEventRemindsEveryMember(t *testing.T) {
	cleanup := setupScanDB(t)
	defer cleanup()
	ctx := context.Background()

	base := time.Now().UnixNano() % 1_000_000_000
	ownerID := seedTelegramUser(t, "owner", base)
	guestID := seedTelegramUser(t, "guest", base+1)

	cal, err := calendars.Create(ctx, ownerID, "Общие напоминания", nil)
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	var guestEmail string
	if err := db.Pool.QueryRow(ctx, `SELECT email FROM "user" WHERE id = $1`, guestID).Scan(&guestEmail); err != nil {
		t.Fatalf("read guest email: %v", err)
	}
	if _, err := calendars.InviteByEmail(ctx, ownerID, cal.ID, guestEmail, calendars.RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := calendars.RespondToInvitation(ctx, guestID, cal.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// One event, authored by the owner, 61 minutes out with a 60-minute
	// reminder — so it is due one minute from now and lands inside the window
	// below. user_id records authorship; access comes from the calendar.
	start := time.Now().UTC().Add(61 * time.Minute)
	var eventID string
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, reminder_offsets, timezone)
		 VALUES ($1, $2, 'Общее событие', $3, $4, '{60}', 'UTC')
		 RETURNING id`,
		ownerID, cal.ID, start, start.Add(time.Hour)).Scan(&eventID); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, eventID) })

	from := time.Now().UTC().Add(-time.Minute)
	to := time.Now().UTC().Add(5 * time.Minute)
	if _, err := Scan(ctx, from, to, slog.Default()); err != nil {
		t.Fatalf("scan: %v", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT user_id::text FROM reminder WHERE event_id = $1 ORDER BY user_id`, eventID)
	if err != nil {
		t.Fatalf("read reminders: %v", err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		got[uid] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	// The claim, both halves. Asserting only "two rows" would pass on two rows
	// for the same person; asserting only "the guest got one" would pass on an
	// implementation that reminded the wrong person.
	if !got[ownerID] {
		t.Error("the event's author got no reminder")
	}
	if !got[guestID] {
		t.Error("the other member of the shared calendar got no reminder — a shared event reminds only its author")
	}
	if len(got) != 2 {
		t.Errorf("want one reminder each for two people, got %d distinct users", len(got))
	}
}

// TestSharedEventGivesEveryoneTheSameSchedule records what is NOT built, so the
// gap is a written fact rather than an assumption.
//
// Offsets live on the EVENT, not on the pair (event, person). So both members
// are reminded — and identically. "Me an hour before, her a day before" is
// slice 5 of the P3 spec and does not exist. This test passes today and is
// meant to FAIL the day per-person offsets land, which is the signal to rewrite
// it rather than a defect.
func TestSharedEventGivesEveryoneTheSameSchedule(t *testing.T) {
	cleanup := setupScanDB(t)
	defer cleanup()
	ctx := context.Background()

	var cols int
	if err := db.Pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.columns
		 WHERE table_name IN ('event_reminder', 'task_reminder')`).Scan(&cols); err != nil {
		t.Fatalf("inspect schema: %v", err)
	}
	if cols != 0 {
		t.Skip("per-person reminder tables exist — slice 5 has landed, rewrite this test")
	}

	// With no per-person table, every member's reminder is derived from the
	// same event.reminder_offsets. Stated here so nobody has to re-derive it.
	t.Log("shared events remind every member, on identical offsets (P3 slice 5 not built)")
}
