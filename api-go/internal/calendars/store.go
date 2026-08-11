package calendars

import (
	"context"

	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB sets the database connection for the calendars package.
func InitDB(d *database.DB) { db = d }

// CalendarIDsFor returns every calendar the user may read.
//
// 🔴 This is meant to become the ONLY sanctioned way to scope a query for
// events or tasks. Task 4 adds scoping_test.go, which fails the build for
// any handler that scopes by user_id instead — that guarantee is not yet
// in effect until that file lands.
//
// Filtering happens in Go rather than SQL so the rule lives in AccessibleIDs,
// where it is tested. The row count here is a handful per user.
func CalendarIDsFor(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT calendar_id::text, status FROM calendar_member WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []Membership{}
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.CalendarID, &m.Status); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return AccessibleIDs(memberships), nil
}

// PersonalIDFor returns the user's own calendar, which is where anything
// created without an explicit calendar goes.
//
// Self-healing: migration 000012 only backfilled a personal calendar for
// users that existed at the time it ran. Nothing creates one for a user who
// registers afterwards (email or Telegram signup both skip it), so on a miss
// this creates it on demand instead of returning pgx.ErrNoRows to the caller.
//
// The fast path checks for calendar AND active membership together, not just
// the calendar. A calendar row can exist with no membership row — e.g. the
// membership insert below failed or the connection dropped between the two
// inserts — and in that state CalendarIDsFor (which only reads
// calendar_member) would silently show the owner none of their own events.
// Checking both closes that state instead of only checking "does the
// calendar exist".
//
// Race-safe: the creation relies on the partial unique index
// idx_calendar_one_personal_per_owner (owner_id WHERE kind = 'personal') from
// Task 1. Two concurrent calls for the same user both attempt the insert;
// exactly one row survives, the other's INSERT is a no-op via ON CONFLICT DO
// NOTHING, and both callers converge on the same calendar id via the
// calendar-only re-read — never two calendars, never an error. Both then
// upsert calendar_member idempotently against its primary key.
func PersonalIDFor(ctx context.Context, userID string) (string, error) {
	id, err := readHealthyPersonalID(ctx, userID)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}

	// Recovery path: either the calendar is missing, or the calendar exists
	// but membership does not. Both are handled by the same sequence below.
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Мой календарь', 'personal')
		 ON CONFLICT DO NOTHING`,
		userID); err != nil {
		return "", err
	}

	// Re-read the calendar alone, without joining on membership: joining here
	// would fail to find the row in the exact "calendar without membership"
	// state this function exists to repair, turning the repair into an error.
	id, err = readCalendarID(ctx, userID)
	if err != nil {
		return "", err
	}

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		id, userID, RoleOwner, StatusActive); err != nil {
		return "", err
	}

	return id, nil
}

// readHealthyPersonalID returns the personal calendar id only when both the
// calendar and an active owner membership exist. pgx.ErrNoRows means either
// piece (or both) is missing.
func readHealthyPersonalID(ctx context.Context, userID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT c.id::text
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id AND m.user_id = c.owner_id
		 WHERE c.owner_id = $1 AND c.kind = 'personal' AND m.status = $2
		 LIMIT 1`,
		userID, StatusActive).Scan(&id)
	return id, err
}

// readCalendarID returns the personal calendar id regardless of whether
// membership exists. Used only inside the recovery path, after the calendar
// row is guaranteed to exist.
func readCalendarID(ctx context.Context, userID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id::text FROM calendar WHERE owner_id = $1 AND kind = 'personal' LIMIT 1`,
		userID).Scan(&id)
	return id, err
}
