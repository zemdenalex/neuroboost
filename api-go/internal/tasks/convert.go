package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/recurrence"
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

// Which part of a repeating task goes into the calendar.
const (
	RepeatSeries = "series"
	RepeatOnce   = "once"
)

var (
	// ErrNeedsTime is returned when a task with no time is asked to become an
	// event. An event without a start is not an event; the caller must ask.
	ErrNeedsTime = errors.New("an event needs a start and an end")

	// ErrUnknownMode is returned for anything that is not move or link.
	ErrUnknownMode = errors.New("mode must be move or link")

	// ErrRepeatChoiceRequired: a repeating task was sent without saying whether
	// the whole series or one day goes. Denis 21.09 — «1 + 2, уточнять»: ask,
	// never guess.
	ErrRepeatChoiceRequired = errors.New("repeat must be series or once for a repeating task")
)

// ConvertRequest is what the caller asks for.
type ConvertRequest struct {
	Mode       string  `json:"mode"`
	Repeat     string  `json:"repeat,omitempty"`
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
//
// A repeating task must say which part goes (spec 21.09, §A1):
//   - series — the event repeats with the task's own rule;
//   - once   — one day goes. With link the event points at the series; with
//     move the day is marked skipped and the series stays (Denis 21.09: «one
//     day moves, the series stays»). Either way the series keeps its reminders.
func Convert(ctx context.Context, userID, taskID string, req ConvertRequest) (*ConvertedEvent, error) {
	if req.Mode != ModeMove && req.Mode != ModeLink {
		return nil, ErrUnknownMode
	}
	if req.Repeat != "" && req.Repeat != RepeatSeries && req.Repeat != RepeatOnce {
		return nil, ErrRepeatChoiceRequired
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
	var rrule *string
	var anchor *time.Time
	var offsets []int
	err = tx.QueryRow(ctx, `
		-- recurrence-agnostic: reads the task to copy it; the series is handled below.
		SELECT title, description, calendar_id::text, COALESCE(tags, '{}'),
		       rrule, repeat_anchor, COALESCE(reminder_offsets, '{}')
		FROM task WHERE id = $1 AND calendar_id = ANY($2)`,
		taskID, calIDs).Scan(&title, &description, &calID, &tags, &rrule, &anchor, &offsets)
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

	tz := userZone(ctx, tx, userID)

	repeating := rrule != nil && *rrule != "" && anchor != nil
	once := repeating && req.Repeat == RepeatOnce
	var eventRule *string
	if repeating {
		if req.Repeat == "" {
			return nil, ErrRepeatChoiceRequired
		}
		rule, perr := recurrence.Parse(*rrule)
		if perr != nil {
			// A pre-18.09 row can hold a rule nothing expands. Copying it would
			// make an event that «repeats» once and never again.
			return nil, fmt.Errorf("%w: %v", ErrInvalidRepeat, perr)
		}
		if once {
			if !recurrence.Occurs(rule, *anchor, LocalDay(startsAt, tz)) {
				return nil, ErrNotAnOccurrence
			}
		} else {
			eventRule = rrule
		}
	}

	link := &taskID
	if req.Mode == ModeMove && once {
		link = nil // a move is not a link; the series stays behind
	}
	ev, err := insertLinkedEvent(ctx, tx, linkedEvent{
		UserID: userID, CalendarID: calID, Title: title, Timezone: tz,
		Description: description, Rrule: eventRule, TaskID: link,
		Tags: tags, ReminderOffsets: offsets,
		StartsAt: startsAt, EndsAt: endsAt, AllDay: req.AllDay,
	})
	if err != nil {
		return nil, err
	}

	if !once {
		// The whole task leaves, so its delivery log follows it. With «once»
		// the series keeps living and keeps its own log.
		//
		// ⚠ There was a DELETE above this on 18.09, guarding against a collision
		// with reminders the event already had. It was DEAD CODE and it is gone:
		// the event is INSERTed just above, inside this very transaction, so
		// nothing can reference it yet. The guard could not fire, and it carried
		// a confident comment explaining why it was necessary — which is the
		// shape that makes dead code survive review.
		//
		// The collision it feared cannot happen either way. The dedupe index
		// (000015:34) already guarantees the task's own reminders are unique per
		// (offset, occurrence); rewriting them all to one event id preserves that.
		if err := handOverReminderLog(ctx, tx, taskID, ev.ID); err != nil {
			return nil, err
		}
	}

	switch {
	case req.Mode == ModeMove && once:
		// The day leaves the series; the series does not leave.
		if _, err := tx.Exec(ctx, `
			INSERT INTO task_occurrence (user_id, task_id, occurrence, state)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (task_id, occurrence)
			DO UPDATE SET state = EXCLUDED.state, acted_at = NOW()`,
			userID, taskID, LocalDay(startsAt, tz).Format("2006-01-02"), StateSkipped); err != nil {
			return nil, err
		}
	case req.Mode == ModeMove:
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
	return ev, nil
}

// linkedEvent is everything an event made from a task carries.
type linkedEvent struct {
	UserID, CalendarID, Title, Timezone string
	Description, Rrule, Color, TaskID   *string
	Tags                                []string
	ReminderOffsets                     []int
	StartsAt, EndsAt                    time.Time
	AllDay                              bool
}

// insertLinkedEvent is the one way a task becomes an event row: Convert and the
// quick «⏰ Запланировать» both come here, so the two paths cannot drift again
// (until 21.09 the quick one copied neither description nor tags, and neither
// path copied reminder_offsets — an event with {} never reminds).
//
// The destination is re-checked with the singular resolver: this function is an
// INSERT, and «the caller already checked» is the sentence scheduleTask fell
// through in August.
func insertLinkedEvent(ctx context.Context, tx pgx.Tx, in linkedEvent) (*ConvertedEvent, error) {
	calID, err := calendars.WritableIDFor(ctx, in.UserID, in.CalendarID)
	if err != nil {
		return nil, err
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	offsets := in.ReminderOffsets
	if offsets == nil {
		offsets = []int{}
	}
	var ev ConvertedEvent
	err = tx.QueryRow(ctx, `
		INSERT INTO event (user_id, calendar_id, title, description, starts_at, ends_at,
		                   all_day, tags, task_id, timezone, rrule, color, reminder_offsets)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id::text, calendar_id::text, title, starts_at, ends_at, all_day, task_id::text`,
		in.UserID, calID, in.Title, in.Description, in.StartsAt, in.EndsAt, in.AllDay,
		tags, in.TaskID, in.Timezone, in.Rrule, in.Color, offsets).
		Scan(&ev.ID, &ev.CalendarID, &ev.Title, &ev.StartsAt, &ev.EndsAt, &ev.AllDay, &ev.TaskID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// handOverReminderLog gives a task's reminder journal to the event it became.
//
// 🔴 Queued rows are dropped, not moved. A PENDING row was timed off the task's
// due DAY; moved under the event it would still fire at that time, next to the
// row the scanner now builds from the event's own start and offsets — measured
// 21.09 with the real scanner: three queued reminders for one thing. Snoozes
// (negative minutes_before) and everything already sent or answered move, so
// history and a pending «remind me in 10 minutes» survive.
func handOverReminderLog(ctx context.Context, tx pgx.Tx, taskID, eventID string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM reminder
		 WHERE task_id = $1 AND status = 'PENDING' AND minutes_before >= 0`, taskID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE reminder
		   SET event_id = $2, task_id = NULL, source_kind = 'EVENT'
		 WHERE task_id = $1`, taskID, eventID)
	return err
}

// rowQuerier is what both the pool and a transaction offer for one row.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// userZone is the zone the user lives in, for storing on what they create.
//
// 🔴 Until 21.09 every event made from a task was stamped 'Europe/Moscow'.
// The instant was still right — starts_at carries its offset — but the zone is
// what the recurrence expander keeps the local hour in, so a New York series
// slid by an hour at the first DST change after it was created.
func userZone(ctx context.Context, q rowQuerier, userID string) string {
	tz := "Europe/Moscow"
	_ = q.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow') FROM "user" WHERE id = $1`, userID).Scan(&tz)
	return tz
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
