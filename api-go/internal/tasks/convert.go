package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
)

// Turning a task into an event, or linking the two.
//
// Настя через Дениса, 17–18.09: ошибся типом — и остаётся создать заново и
// удалить старое. Denis chose both modes («1 + 3»): either MOVE it, or ADD an
// event and keep the task linked to it.
//
// 🔴 The link is `event.task_id`, which has existed since the baseline
// (000001_baseline.up.sql:129) with ON DELETE SET NULL — exactly right for
// «move», where the task disappears and the event must survive. I briefly added
// a second column pointing the other way; 000018 removed it. Two columns for one
// relationship drift apart.

// Convert modes.
const (
	ModeMove = "move"
	ModeLink = "link"
)

var (
	// ErrNeedsTime is returned when a task with no time is asked to become an
	// event. An event without a start is not an event; the caller must ask.
	ErrNeedsTime = errors.New("an event needs a start and an end")

	// ErrUnknownMode is returned for anything that is not move or link.
	ErrUnknownMode = errors.New("mode must be move or link")
)

// ConvertRequest is what the caller asks for.
type ConvertRequest struct {
	Mode       string  `json:"mode"`
	StartsAt   string  `json:"starts_at"`
	EndsAt     string  `json:"ends_at"`
	AllDay     bool    `json:"all_day,omitempty"`
	CalendarID *string `json:"calendar_id,omitempty"`
}

// ConvertedEvent is the event that came out of a task.
type ConvertedEvent struct {
	ID         string    `json:"id"`
	CalendarID string    `json:"calendar_id"`
	Title      string    `json:"title"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	AllDay     bool      `json:"all_day"`
	TaskID     *string   `json:"task_id,omitempty"`
}

// Convert turns a task into an event, in one transaction.
//
// 🔴 One transaction because a half-done conversion is the worst outcome
// available: an event created and a task left behind is a duplicate the user has
// to notice and clean up, and reminders moved to an event that was never created
// are reminders that never arrive.
func Convert(ctx context.Context, userID, taskID string, req ConvertRequest) (*ConvertedEvent, error) {
	if req.Mode != ModeMove && req.Mode != ModeLink {
		return nil, ErrUnknownMode
	}
	if req.StartsAt == "" || req.EndsAt == "" {
		// Refused rather than guessed. A task carries a due DAY, not a time, and
		// inventing 09:00–10:00 would put a thing in the calendar at an hour
		// nobody chose — then remind about it.
		return nil, ErrNeedsTime
	}
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, fmt.Errorf("%w: bad starts_at", ErrNeedsTime)
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, fmt.Errorf("%w: bad ends_at", ErrNeedsTime)
	}
	if !endsAt.After(startsAt) {
		return nil, fmt.Errorf("%w: end is not after start", ErrNeedsTime)
	}

	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Rollback is a no-op once the tx is committed, so this is safe on every
	// path and covers the returns in between.
	defer func() { _ = tx.Rollback(ctx) }()

	var title string
	var description *string
	var calID string
	var tags []string
	err = tx.QueryRow(ctx, `
		SELECT title, description, calendar_id::text, COALESCE(tags, '{}')
		FROM task WHERE id = $1 AND calendar_id = ANY($2)`,
		taskID, calIDs).Scan(&title, &description, &calID, &tags)
	if err != nil {
		return nil, err
	}

	// 🔴 The DESTINATION is checked with the singular resolver, which answers
	// "may I write in THIS calendar" — not with the plural list, which only
	// scopes a WHERE.
	//
	// The distinction is not pedantry: it is the exact hole scheduleTask fell
	// through, and calendars/writescoping_test.go caught this function the first
	// time it ran. My first version checked the destination against the plural
	// list by hand, which works today and is the shape that stopped working the
	// moment somebody trusted it.
	if req.CalendarID != nil && *req.CalendarID != "" {
		dest, derr := calendars.WritableIDFor(ctx, userID, *req.CalendarID)
		if derr != nil {
			return nil, derr
		}
		calID = dest
	}

	var ev ConvertedEvent
	err = tx.QueryRow(ctx, `
		INSERT INTO event (user_id, calendar_id, title, description, starts_at, ends_at,
		                   all_day, tags, task_id, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'Europe/Moscow')
		RETURNING id::text, calendar_id::text, title, starts_at, ends_at, all_day, task_id::text`,
		userID, calID, title, description, startsAt, endsAt, req.AllDay, tags, taskID).
		Scan(&ev.ID, &ev.CalendarID, &ev.Title, &ev.StartsAt, &ev.EndsAt, &ev.AllDay, &ev.TaskID)
	if err != nil {
		return nil, err
	}

	// Reminders follow the thing they are about.
	//
	// ⚠ There was a DELETE above this on 18.09, guarding against a collision
	// with reminders the event already had. It was DEAD CODE and it is gone:
	// the event is INSERTed three statements earlier, inside this very
	// transaction, so nothing can reference it yet. The guard could not fire,
	// and it carried a confident comment explaining why it was necessary —
	// which is the shape that makes dead code survive review.
	//
	// The collision it feared cannot happen either way. The dedupe index
	// (000015:34) already guarantees the task's own reminders are unique per
	// (offset, occurrence); rewriting them all to one event id preserves that.
	if _, err := tx.Exec(ctx, `
		UPDATE reminder
		   SET event_id = $2, task_id = NULL, source_kind = 'EVENT'
		 WHERE task_id = $1`, taskID, ev.ID); err != nil {
		return nil, err
	}

	if req.Mode == ModeMove {
		// 🔴 event.task_id is ON DELETE SET NULL, so deleting the task leaves
		// the event standing with a null link. That is the intended end state
		// for a move: one thing, in the calendar.
		if _, err := tx.Exec(ctx,
			`DELETE FROM task WHERE id = $1 AND calendar_id = ANY($2)`, taskID, calIDs); err != nil {
			return nil, err
		}
		ev.TaskID = nil
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &ev, nil
}

// ConvertedFrom reports the event a task was linked to, if any.
func ConvertedFrom(ctx context.Context, taskID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id::text FROM event WHERE task_id = $1 LIMIT 1`, taskID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}
