package calendars

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// A second user, so the calendar under test is genuinely someone else's.
func seedUser(t *testing.T, ctx context.Context) (userID string, cleanup func()) {
	t.Helper()
	email := fmt.Sprintf("writable-ids-other-%d@example.com", time.Now().UnixNano())
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("seed second user: %v", err)
	}
	return userID, func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
	}
}

// WritableIDsFor is what stops read access from becoming write access.
//
// 🔴 These tests need a real database and will SKIP without DATABASE_URL, which
// CI sets. That is a known hazard in this repository — two tests skipped
// silently on 2026-08-14 and one of them had already been reported as covering
// its case — so the assertions below are written to fail loudly rather than to
// pass on an empty result, and the CI log is the evidence, not a local run.

// Inserts a membership directly. Nothing in the product creates a viewer today
// (invitations are slice 3), so the row the check exists for has to be made
// here or the test would assert against a case that cannot occur.
func addMember(t *testing.T, ctx context.Context, calendarID, userID, role string) {
	t.Helper()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)`,
		calendarID, userID, role, StatusActive); err != nil {
		t.Fatalf("seed %s membership: %v", role, err)
	}
}

func TestWritableIDsForIncludesOwnedAndEditableCalendars(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	own, err := Create(ctx, userID, "Owned", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// A calendar someone else owns, on which this user is an editor.
	otherID, otherCleanup := seedUser(t, ctx)
	defer otherCleanup()
	shared, err := Create(ctx, otherID, "Shared", nil)
	if err != nil {
		t.Fatalf("create shared: %v", err)
	}
	addMember(t, ctx, shared.ID, userID, RoleEditor)

	ids, err := WritableIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("writable ids: %v", err)
	}

	if !contains(ids, own.ID) {
		t.Errorf("a calendar the user owns is missing from the writable set")
	}
	if !contains(ids, shared.ID) {
		t.Errorf("a calendar the user edits is missing from the writable set — editors could no longer write")
	}
}

// The case the function exists for. Read access must not carry write access.
func TestWritableIDsForExcludesAViewerCalendar(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	otherID, otherCleanup := seedUser(t, ctx)
	defer otherCleanup()
	readOnly, err := Create(ctx, otherID, "Read only", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	addMember(t, ctx, readOnly.ID, userID, RoleViewer)

	readable, err := CalendarIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("readable ids: %v", err)
	}
	// The positive control: without this, "not writable" could simply mean the
	// membership was never created and the test would pass having proved
	// nothing at all.
	if !contains(readable, readOnly.ID) {
		t.Fatalf("the viewer membership was not created — the exclusion below would prove nothing")
	}

	writable, err := WritableIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("writable ids: %v", err)
	}
	if contains(writable, readOnly.ID) {
		t.Errorf("a viewer's calendar is writable: read access would let them edit, move and delete other people's events")
	}
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
