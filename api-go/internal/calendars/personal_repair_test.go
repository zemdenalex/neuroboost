package calendars

import (
	"context"
	"testing"
)

// PersonalIDFor promises to REPAIR a broken personal calendar, and until
// 2026-08-16 it left two broken states exactly as it found them, because its
// membership upsert said ON CONFLICT DO NOTHING.
//
// Both states are reachable the moment anything writes a membership row for a
// personal calendar in the wrong shape. Neither is reachable through today's
// product — which is why they are worth a test rather than a note: nothing else
// will notice when they become reachable.
//
// The assertion is deliberately about what the USER can do afterwards
// (CalendarIDsFor / WritableIDsFor), not about the row: PersonalIDFor returned
// the right id in the broken states too, so an assertion on its return value
// would have passed while the user saw none of their own events.

func TestPersonalIDForRepairsAnInvitedMembership(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	id, err := PersonalIDFor(ctx, userID)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// The broken state: membership exists but was never accepted.
	if _, err := db.Pool.Exec(ctx,
		`UPDATE calendar_member SET status = $1 WHERE calendar_id = $2 AND user_id = $3`,
		StatusInvited, id, userID); err != nil {
		t.Fatalf("break the membership: %v", err)
	}
	readable, err := CalendarIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("readable: %v", err)
	}
	// Positive control: without this, "repaired" could mean the break never took.
	if contains(readable, id) {
		t.Fatalf("the membership was not actually broken — the repair below would prove nothing")
	}

	if _, err := PersonalIDFor(ctx, userID); err != nil {
		t.Fatalf("repair call: %v", err)
	}

	readable, err = CalendarIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("readable after repair: %v", err)
	}
	if !contains(readable, id) {
		t.Error("after the repair the user still cannot see their own personal calendar")
	}
}

func TestPersonalIDForRepairsANonOwnerMembership(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	id, err := PersonalIDFor(ctx, userID)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// The broken state: the owner's own membership says viewer, so
	// WritableIDsFor excludes their personal calendar and they cannot create
	// anything in it.
	if _, err := db.Pool.Exec(ctx,
		`UPDATE calendar_member SET role = $1 WHERE calendar_id = $2 AND user_id = $3`,
		RoleViewer, id, userID); err != nil {
		t.Fatalf("break the role: %v", err)
	}
	writable, err := WritableIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("writable: %v", err)
	}
	if contains(writable, id) {
		t.Fatalf("the role was not actually broken — the repair below would prove nothing")
	}

	if _, err := PersonalIDFor(ctx, userID); err != nil {
		t.Fatalf("repair call: %v", err)
	}

	writable, err = WritableIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("writable after repair: %v", err)
	}
	if !contains(writable, id) {
		t.Error("after the repair the user still cannot write to their own personal calendar")
	}
}
