package calendars

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pgErrCode reports whether err is a *pgconn.PgError with the given SQLSTATE.
func pgErrCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// maxNameLen bounds the stored name. The column is TEXT; this is a product
// limit, not a storage one.
const maxNameLen = 100

// ListFor returns every calendar the user belongs to, personal one first.
//
// It calls PersonalIDFor before reading so a user who registered after
// migration 000012 ran sees their personal calendar on the very first request
// instead of an empty list — that self-healing lives in one place and this is
// the first read path that would expose its absence.
//
// Deliberately unlike AccessibleIDs and membership: this does NOT filter by
// m.status. An invited-but-not-yet-accepted membership is included, with
// Status == StatusInvited on the wire — that is how a caller (and slice 3's
// invitation UI) tells an invitation apart from active membership. Being
// listed here does not imply being readable: CalendarIDsFor, which does
// filter by status, remains the only access rule for event/task scoping.
func ListFor(ctx context.Context, userID string) ([]Calendar, error) {
	if _, err := PersonalIDFor(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT c.id::text, c.name, c.color, c.kind, m.role, m.status, c.created_at
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id
		 WHERE m.user_id = $1
		 ORDER BY (c.kind = 'personal') DESC, c.created_at`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Calendar{}
	for rows.Next() {
		var c Calendar
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.Role, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// NormalizeName trims a supplied name and reports whether it is usable.
// Exported so the handler validates with the same rule the store enforces.
func NormalizeName(name string) (string, bool) {
	n := strings.TrimSpace(name)
	if n == "" || len([]rune(n)) > maxNameLen {
		return "", false
	}
	return n, true
}

// Create makes a shared calendar and makes its creator the owning member.
//
// Both inserts run in one transaction: a calendar with no membership row is
// invisible to its own creator (CalendarIDsFor reads calendar_member only),
// and nothing would ever repair it — PersonalIDFor only heals the personal one.
func Create(ctx context.Context, userID, name string, color *string) (Calendar, error) {
	n, ok := NormalizeName(name)
	if !ok {
		return Calendar{}, ErrInvalidName
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return Calendar{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var c Calendar
	if err := tx.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, color, kind)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, name, color, kind, created_at`,
		userID, n, color, KindShared,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.CreatedAt); err != nil {
		return Calendar{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)`,
		c.ID, userID, RoleOwner, StatusActive); err != nil {
		return Calendar{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Calendar{}, err
	}

	c.Role = RoleOwner
	c.Status = StatusActive
	return c, nil
}

// membership reads the caller's own row for a calendar. pgx.ErrNoRows here
// means "not a member", which callers translate to ErrCalendarNotFound.
//
// A malformed calendarID (not a valid UUID) makes Postgres reject the value
// while coercing $1 to uuid — SQLSTATE 22P02, invalid_text_representation.
// That is normalized to pgx.ErrNoRows here rather than left as a raw driver
// error, so a junk id in a URL path reads as "no such calendar" (the existing
// ErrNoRows handling in requireOwner) instead of a 500.
func membership(ctx context.Context, userID, calendarID string) (kind, role string, err error) {
	err = db.Pool.QueryRow(ctx,
		`SELECT c.kind, m.role
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id
		 WHERE c.id = $1 AND m.user_id = $2 AND m.status = $3`,
		calendarID, userID, StatusActive).Scan(&kind, &role)
	if err != nil && pgErrCode(err, "22P02") {
		return "", "", pgx.ErrNoRows
	}
	return kind, role, err
}

// requireOwner resolves the caller's standing in one place so update and
// delete cannot drift apart.
func requireOwner(ctx context.Context, userID, calendarID string) (kind string, err error) {
	kind, role, err := membership(ctx, userID, calendarID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrCalendarNotFound
	}
	if err != nil {
		return "", err
	}
	if role != RoleOwner {
		return "", ErrNotCalendarOwner
	}
	return kind, nil
}

// Update renames or recolours a calendar. Owner only, including the personal
// one — renaming "Мой календарь" is harmless and expected.
func Update(ctx context.Context, userID, calendarID string, p UpdateFields) (Calendar, error) {
	if _, err := requireOwner(ctx, userID, calendarID); err != nil {
		return Calendar{}, err
	}

	if p.Name != nil {
		n, ok := NormalizeName(*p.Name)
		if !ok {
			return Calendar{}, ErrInvalidName
		}
		p.Name = &n
	}

	var c Calendar
	if err := db.Pool.QueryRow(ctx,
		// COALESCE cannot express "set to null" — that is exactly what it is
		// for — so clearing takes its own branch rather than a magic value.
		`UPDATE calendar
		 SET name  = COALESCE($2, name),
		     color = CASE WHEN $4 THEN NULL ELSE COALESCE($3, color) END
		 WHERE id = $1
		 RETURNING id::text, name, color, kind, created_at`,
		calendarID, p.Name, p.Color, p.ClearColor,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.CreatedAt); err != nil {
		// Can only happen if the row disappeared between the pre-flight
		// requireOwner check above and this UPDATE (e.g. a concurrent delete).
		if errors.Is(err, pgx.ErrNoRows) {
			return Calendar{}, ErrCalendarNotFound
		}
		return Calendar{}, err
	}

	c.Role = RoleOwner
	c.Status = StatusActive
	return c, nil
}

// Delete removes an empty shared calendar.
//
// Two refusals, both load-bearing:
//   - personal calendars are never deletable (see ErrCalendarIsPersonal);
//   - a non-empty calendar reports its contents instead of cascading. Deleting
//     it would take both members' events, including ones the deleter did not
//     create (spec §5.1).
func Delete(ctx context.Context, userID, calendarID string) error {
	kind, err := requireOwner(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if kind == KindPersonal {
		return ErrCalendarIsPersonal
	}

	var ne NotEmptyError
	if err := db.Pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM event WHERE calendar_id = $1),
		        (SELECT count(*) FROM task  WHERE calendar_id = $1)`,
		calendarID).Scan(&ne.Events, &ne.Tasks); err != nil {
		return err
	}
	if ne.Events > 0 || ne.Tasks > 0 {
		return &ne
	}

	// calendar_member cascades from calendar; nothing else references an empty
	// calendar at this point — except a TOCTOU race with a concurrent insert
	// into event/task between the count above and this DELETE, which the FK
	// (no ON DELETE on event.calendar_id / task.calendar_id) turns into
	// SQLSTATE 23503 instead of silently dropping the new row's calendar.
	_, err = db.Pool.Exec(ctx, `DELETE FROM calendar WHERE id = $1`, calendarID)
	if err != nil && pgErrCode(err, "23503") {
		var fresh NotEmptyError
		if err2 := db.Pool.QueryRow(ctx,
			`SELECT (SELECT count(*) FROM event WHERE calendar_id = $1),
			        (SELECT count(*) FROM task  WHERE calendar_id = $1)`,
			calendarID).Scan(&fresh.Events, &fresh.Tasks); err2 != nil {
			return err2
		}
		if fresh.Events > 0 || fresh.Tasks > 0 {
			return &fresh
		}
		// The race resolved to something else the FK caught (not a leftover
		// event/task) — surface the original driver error rather than
		// inventing an empty NotEmptyError that would misreport as 409.
		return err
	}
	return err
}

// TransferOwnership hands a shared calendar to another active member.
//
// 🔴 This is the ONLY sanctioned way to change an owner, and the reason is that
// ownership is written in two places. `calendar_member.role = 'owner'` is the
// source of truth — requireOwner reads it and nothing else. `calendar.owner_id`
// is a denormalised cache of the same fact.
//
// The cache is not decorative and cannot simply be dropped:
// migrations/000012_calendars.up.sql:10 declares it `NOT NULL REFERENCES
// "user"(id) ON DELETE CASCADE`, so it is what ties a calendar's lifetime to a
// real user row. Removing it means a migration plus a decision about what
// happens to a shared calendar whose creator deletes their account — separate
// work, deliberately not done here.
//
// So both rows move, in ONE transaction. Update them separately and the two
// disagree invisibly: requireOwner says "you are the owner" while
// PersonalIDFor and readCalendarID say otherwise, and deleting the stale user
// starts failing on calendars they no longer own.
//
// Personal calendars cannot be transferred: a personal calendar IS its owner.
func TransferOwnership(ctx context.Context, currentOwnerID, calendarID, newOwnerID string) error {
	kind, err := requireOwner(ctx, currentOwnerID, calendarID)
	if err != nil {
		return err
	}
	if kind == KindPersonal {
		return ErrCalendarIsPersonal
	}
	if newOwnerID == currentOwnerID {
		// Not an error worth a distinct code: the requested end state already
		// holds, and doing the writes would demote the owner to editor and back.
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The recipient must already be an active member. Promoting a stranger
	// would be an invitation, which is a different operation with a different
	// consent story.
	var newRole string
	if err := tx.QueryRow(ctx,
		`SELECT role FROM calendar_member
		 WHERE calendar_id = $1 AND user_id = $2 AND status = $3`,
		calendarID, newOwnerID, StatusActive).Scan(&newRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || pgErrCode(err, "22P02") {
			return ErrCalendarNotFound
		}
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE calendar_member SET role = $1
		 WHERE calendar_id = $2 AND user_id = $3`,
		RoleEditor, calendarID, currentOwnerID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE calendar_member SET role = $1
		 WHERE calendar_id = $2 AND user_id = $3`,
		RoleOwner, calendarID, newOwnerID); err != nil {
		return err
	}

	// The cache, in the same transaction as the truth it caches.
	if _, err := tx.Exec(ctx,
		`UPDATE calendar SET owner_id = $1 WHERE id = $2`,
		newOwnerID, calendarID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
