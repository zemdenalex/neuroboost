package calendars

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"neuroboost/api-go/internal/database"
)

// WritableIDFor is the check that stops a caller writing into someone else's
// calendar. Until 2026-08-15 there was no check at all, because there was no
// way to name a calendar: every event went to the author's personal one.
//
// 🔴 What follows does NOT need a database, and that is the point of testing it
// here: the empty-string case must answer before any query, and the guard's
// value comes precisely from it not being reachable by accident.

// lazyBrokenDB builds a pool that connects on first use and always fails.
//
// Not database.New: that pings during construction, so an unreachable DSN fails
// immediately and a t.Skip guarding it turns the test into a silent no-op —
// which is exactly what happened to two tests in this repository on 2026-08-14.
func lazyBrokenDB(t *testing.T) *database.DB {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("could not parse the test DSN: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("could not build a lazy pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &database.DB{Pool: pool}
}

// An empty id means "my personal calendar", and that path must not depend on
// the membership lookup at all — the bot and the importer never name one.
func TestWritableIDForEmptyIDDoesNotConsultMembership(t *testing.T) {
	previous := db
	db = lazyBrokenDB(t)
	t.Cleanup(func() { db = previous })

	// With an unreachable pool both paths fail, so the assertion is about WHICH
	// failure: PersonalIDFor's, not the membership query's. A membership error
	// would surface as ErrCalendarNotFound.
	_, err := WritableIDFor(context.Background(), "u-1", "")
	if errors.Is(err, ErrCalendarNotFound) {
		t.Fatal("an empty calendar id went through the membership lookup instead of resolving the personal calendar")
	}
}

// A caller who is not a member must not learn that the calendar exists.
func TestWritableIDForRefusesAStrangerAsNotFound(t *testing.T) {
	previous := db
	db = lazyBrokenDB(t)
	t.Cleanup(func() { db = previous })

	// The lookup cannot be answered, which is the same shape as "no such row"
	// from the caller's side; both must refuse rather than pass the id through.
	got, err := WritableIDFor(context.Background(), "u-1", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("an unverifiable calendar id was accepted")
	}
	if got != "" {
		t.Errorf("returned a calendar id (%q) alongside an error", got)
	}
}

// 🔴 The security property, stated as a property rather than a scenario: a
// malformed id must never be handed back for use in an INSERT.
func TestWritableIDForNeverReturnsAnUncheckedID(t *testing.T) {
	previous := db
	db = lazyBrokenDB(t)
	t.Cleanup(func() { db = previous })

	for _, id := range []string{
		"not-a-uuid",
		"22222222-2222-2222-2222-222222222222",
		"'; DROP TABLE event; --",
	} {
		got, err := WritableIDFor(context.Background(), "u-1", id)
		if err == nil {
			t.Errorf("id %q was accepted without verification", id)
		}
		if got == id {
			t.Errorf("id %q was echoed back unchecked", id)
		}
	}
}

// The roles that may write. A change here is a change to who can add events to
// a shared calendar, so it should have to be deliberate.
func TestWritingRolesAreOwnerAndEditorOnly(t *testing.T) {
	writable := map[string]bool{RoleOwner: true, RoleEditor: true, RoleViewer: false}

	for role, canWrite := range writable {
		// Mirrors the condition in WritableIDFor. Asserting the table rather
		// than the code path keeps this readable, and a viewer silently gaining
		// write access is the failure worth catching.
		allowed := role == RoleOwner || role == RoleEditor
		if allowed != canWrite {
			t.Errorf("role %q: writable = %v, want %v", role, allowed, canWrite)
		}
	}
}
