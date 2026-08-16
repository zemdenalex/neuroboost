package calendars

import (
	"context"
	"errors"

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

// WritableIDsFor returns every calendar the user may CHANGE things in.
//
// 🔴 The distinction from CalendarIDsFor is the whole point, and getting it
// wrong in either direction is a defect:
//
//   - Scoping a WRITE by CalendarIDsFor lets a viewer edit, move, resize and
//     delete events in a calendar they were given read access to. Nothing
//     downstream objects, because the row's user_id records who wrote the
//     event, not who may change it.
//   - Scoping a READ by this function would hide a shared calendar's events
//     from the very people it was shared with, and stop their reminders
//     (reminders/scan.go scopes by CalendarIDsFor on purpose).
//
// Reads keep CalendarIDsFor. Writes use this. Today the difference is latent —
// nothing creates a viewer membership yet, since invitations are slice 3 —
// but "latent" is a property of the calling code, not of this rule, and the
// day it stops being latent is the day someone is handed read access.
//
// Filtering is in SQL rather than in AccessibleIDs because the role is what is
// being filtered on, and a Go-side filter would have to fetch the role column
// that AccessibleIDs deliberately does not carry.
func WritableIDsFor(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT calendar_id::text
		   FROM calendar_member
		  WHERE user_id = $1 AND status = $2 AND role = ANY($3)`,
		userID, StatusActive, []string{RoleOwner, RoleEditor})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
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

	// 🔴 DO UPDATE, not DO NOTHING. This function's whole promise is that it
	// REPAIRS a broken personal calendar, and DO NOTHING left two broken states
	// exactly as it found them: a membership row stuck at status 'invited', or
	// one whose role is not owner. In either case readHealthyPersonalID keeps
	// failing, this branch keeps doing nothing, and PersonalIDFor still returns
	// the id — so the caller believes it is repaired while CalendarIDsFor,
	// which filters on status, shows the user none of their own events.
	//
	// Safe to force here and nowhere else: this is the owner's OWN personal
	// calendar, and owner+active is the only state it may be in.
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (calendar_id, user_id)
		 DO UPDATE SET role = EXCLUDED.role, status = EXCLUDED.status`,
		id, userID, RoleOwner, StatusActive); err != nil {
		return "", err
	}

	return id, nil
}

// readHealthyPersonalID returns the personal calendar id only when both the
// calendar and an active OWNER membership exist. pgx.ErrNoRows means either
// piece (or both) is missing, or the membership is there in the wrong shape.
//
// 🔴 The role check matters as much as the status one. WritableIDsFor filters
// on role, so an owner whose own membership said 'viewer' could read their
// personal calendar and not write to it — and without this clause PersonalIDFor
// would call that healthy and repair nothing.
func readHealthyPersonalID(ctx context.Context, userID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT c.id::text
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id AND m.user_id = c.owner_id
		 WHERE c.owner_id = $1 AND c.kind = 'personal'
		   AND m.status = $2 AND m.role = $3
		 LIMIT 1`,
		userID, StatusActive, RoleOwner).Scan(&id)
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

// WritableIDFor resolves the calendar an event or task should be created in.
//
// Absent calendarID means the caller's personal calendar — the behaviour every
// creation had before shared calendars existed, and the one the bot and import
// still rely on.
//
// 🔴 A supplied id is CHECKED, not trusted. Without this, a caller could post
// any UUID and write into someone else's calendar: the row's user_id records
// authorship, not access, so nothing downstream would object.
//
// A viewer gets ErrNotCalendarOwner — they can read the calendar but not add to
// it. A non-member gets ErrCalendarNotFound rather than a permission error, for
// the same reason requireOwner does: a 403 would confirm to a stranger that the
// calendar exists.
func WritableIDFor(ctx context.Context, userID, calendarID string) (string, error) {
	if calendarID == "" {
		return PersonalIDFor(ctx, userID)
	}

	var role string
	err := db.Pool.QueryRow(ctx,
		`SELECT m.role
		   FROM calendar_member m
		  WHERE m.calendar_id = $1 AND m.user_id = $2 AND m.status = $3`,
		calendarID, userID, StatusActive).Scan(&role)
	if err != nil {
		// 22P02: the id was not a UUID at all. Reads as "no such calendar"
		// rather than a 500, matching membership() in crud.go.
		if errors.Is(err, pgx.ErrNoRows) || pgErrCode(err, "22P02") {
			return "", ErrCalendarNotFound
		}
		return "", err
	}

	if role != RoleOwner && role != RoleEditor {
		return "", ErrNotCalendarOwner
	}
	return calendarID, nil
}
