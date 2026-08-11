package calendars

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// maxNameLen bounds the stored name. The column is TEXT; this is a product
// limit, not a storage one.
const maxNameLen = 100

// ListFor returns every calendar the user belongs to, personal one first.
//
// It calls PersonalIDFor before reading so a user who registered after
// migration 000012 ran sees their personal calendar on the very first request
// instead of an empty list — that self-healing lives in one place and this is
// the first read path that would expose its absence.
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
		return Calendar{}, errors.New("invalid name")
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
func membership(ctx context.Context, userID, calendarID string) (kind, role string, err error) {
	err = db.Pool.QueryRow(ctx,
		`SELECT c.kind, m.role
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id
		 WHERE c.id = $1 AND m.user_id = $2 AND m.status = $3`,
		calendarID, userID, StatusActive).Scan(&kind, &role)
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
			return Calendar{}, errors.New("invalid name")
		}
		p.Name = &n
	}

	var c Calendar
	if err := db.Pool.QueryRow(ctx,
		`UPDATE calendar
		 SET name  = COALESCE($2, name),
		     color = COALESCE($3, color)
		 WHERE id = $1
		 RETURNING id::text, name, color, kind, created_at`,
		calendarID, p.Name, p.Color,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.CreatedAt); err != nil {
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
	// calendar at this point.
	_, err = db.Pool.Exec(ctx, `DELETE FROM calendar WHERE id = $1`, calendarID)
	return err
}
