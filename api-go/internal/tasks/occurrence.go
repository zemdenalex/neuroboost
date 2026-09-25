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

// Per-day state of a repeating task.
//
// Denis, 18.09, on what a repeating task is:
//
//	«not totally 1 task, but not totally different tasks… more like a big task
//	 with subtasks, but I think it should be it's kind of own entity»
//
// 🔴 The state of a DAY never touches task.status. Ticking today's pills through
// status = 'DONE' would take the series out of every list for ever, which is the
// one failure this whole shape exists to prevent. `status` describes the series
// (TODO = running, DONE = switched off); a row here describes one day of it.
//
// Sparse: a row exists only for a day something happened to. A year of daily
// pills is as many rows as times a button was pressed, not 365.

// Occurrence states.
const (
	StateDone    = "done"
	StateSkipped = "skipped"
	// StateOpen is not stored: it removes the day's row, putting the day
	// back to «no answer» (the web's Undo after a tick, 4.11).
	StateOpen = "open"
)

var (
	// ErrNotRecurring is returned for a task with no repeat rule: a day of a
	// series that does not exist cannot be marked.
	ErrNotRecurring = errors.New("task does not repeat")

	// ErrNotAnOccurrence is returned when the day is not in the series —
	// a Saturday on a weekly Friday task. Refused rather than stored, or the
	// table would fill with days nothing will ever ask about.
	ErrNotAnOccurrence = errors.New("that day is not in the series")
)

// repeatOf reads the rule and anchor of a task the caller may see.
func repeatOf(ctx context.Context, userID, taskID string) (*recurrence.Rule, time.Time, string, error) {
	// 🔴 WritableIDsFor, not CalendarIDsFor: marking a day is a write, and a
	// read-only member of a shared calendar must not be able to tick off
	// somebody else's series.
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, time.Time{}, "", err
	}

	var rrule *string
	var anchor *time.Time
	var tz string
	err = db.Pool.QueryRow(ctx, `
		SELECT t.rrule, t.repeat_anchor, COALESCE(u.timezone, 'Europe/Moscow')
		  FROM task t
		  JOIN "user" u ON u.id = $3
		 WHERE t.id = $1 AND t.calendar_id = ANY($2)`,
		taskID, calIDs, userID).Scan(&rrule, &anchor, &tz)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	if rrule == nil || *rrule == "" || anchor == nil {
		return nil, time.Time{}, tz, ErrNotRecurring
	}

	rule, err := recurrence.Parse(*rrule)
	if err != nil {
		// A stored rule this API cannot parse is a series nobody can act on.
		// Since 18.09 writes are refused at the door, so this can only be an
		// older row — and it must say so rather than behave as a one-off.
		return nil, time.Time{}, tz, fmt.Errorf("stored rule is unsupported: %w", err)
	}
	return rule, *anchor, tz, nil
}

// LocalDay is the calendar day an instant falls on, where the user is.
//
// 🔴 In the USER's zone, never the server's. The API runs in UTC, so without
// this a «done» tapped at 23:00 in Tokyo — or 00:30 in Moscow — is filed under
// the wrong day and the streak breaks for reasons nobody can see. This is the
// same mistake that made a due_date read a day early on 17.09; that one was only
// a misread, this one would be stored.
func LocalDay(at time.Time, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	l := at.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// OccurrenceState reports what was done with one day: "", "done" or "skipped".
func OccurrenceState(ctx context.Context, taskID string, day time.Time) (string, error) {
	var state string
	err := db.Pool.QueryRow(ctx,
		`SELECT state FROM task_occurrence WHERE task_id = $1 AND occurrence = $2`,
		taskID, day.Format("2006-01-02")).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return state, nil
}

// MarkOccurrence records what happened to one day of a series.
func MarkOccurrence(ctx context.Context, userID, taskID string, day time.Time, state string) error {
	if state != StateDone && state != StateSkipped && state != StateOpen {
		return fmt.Errorf("unknown occurrence state: %s", state)
	}

	rule, anchor, _, err := repeatOf(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if !recurrence.Occurs(rule, anchor, day) {
		return ErrNotAnOccurrence
	}

	if state == StateOpen {
		_, err = db.Pool.Exec(ctx,
			`DELETE FROM task_occurrence WHERE task_id = $1 AND user_id = $2 AND occurrence = $3`,
			taskID, userID, day.Format("2006-01-02"))
		return err
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO task_occurrence (user_id, task_id, occurrence, state)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, occurrence)
		DO UPDATE SET state = EXCLUDED.state, acted_at = NOW()`,
		userID, taskID, day.Format("2006-01-02"), state)
	return err
}

// PressedDay answers which day «✅ готово» means when the caller named no date.
//
// 🔴 Not always today, and that cost Denis the 21.09 pass. A task created
// «позвонить в банк ЗАВТРА … каждый день» anchors its series on tomorrow, so
// today is genuinely not in it — and the button answered him with
// `API error 400: NOT_AN_OCCURRENCE`. The arithmetic was right and the answer
// was useless: he pressed the only button the card offered.
//
// A press means «the one I am looking at»: today when today is in the series,
// otherwise the first day that is. A COUNT or UNTIL series eventually has no
// such day, and only then is the refusal the honest reply.
//
// ⚠ Only for a press. A date the caller NAMED is still refused when it is not
// an occurrence — that guard is what keeps task_occurrence free of days nothing
// will ever ask about, and silently sliding someone's chosen date to a
// neighbouring one would be a worse answer than «no».
func PressedDay(rule *recurrence.Rule, anchor, today time.Time) (time.Time, bool) {
	if rule == nil {
		return time.Time{}, false
	}
	if recurrence.Occurs(rule, anchor, today) {
		return today, true
	}
	return recurrence.Next(rule, anchor, today)
}

// PostponeSeries closes the next `days` days of a series, leaving its rhythm
// untouched.
//
// 🔴 Denis chose this over shifting the series, 18.09: «Пропустить закрытые дни,
// ритм не трогать». Missing a day of a course of pills does not move the course;
// it means those days did not happen. Shifting would also quietly change what
// every future reminder is for.
func PostponeSeries(ctx context.Context, userID, taskID string, from time.Time, days int) (int, error) {
	if days < 1 {
		return 0, fmt.Errorf("postpone needs at least one day, got %d", days)
	}
	rule, anchor, _, err := repeatOf(ctx, userID, taskID)
	if err != nil {
		return 0, err
	}

	closed := 0
	day := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	for i := 0; i < days; i++ {
		if recurrence.Occurs(rule, anchor, day) {
			if _, err := db.Pool.Exec(ctx, `
				INSERT INTO task_occurrence (user_id, task_id, occurrence, state)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (task_id, occurrence)
				DO UPDATE SET state = EXCLUDED.state, acted_at = NOW()`,
				userID, taskID, day.Format("2006-01-02"), StateSkipped); err != nil {
				return closed, err
			}
			closed++
		}
		day = day.AddDate(0, 0, 1)
	}
	return closed, nil
}

// OccurrenceRow is one answered day of a series.
type OccurrenceRow struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"`
	State      string `json:"state"`
}

var (
	// ErrRangeTooLarge bounds a read that statistics repeats per year for «Всё».
	ErrRangeTooLarge = errors.New("range must be at most 366 days")
	// ErrInvalidRange is a range that ends before it starts.
	ErrInvalidRange = errors.New("from must be a date not after to")
)

// ListOccurrences returns the answered days of every series the caller can see
// (spec 22.09 §5 — statistics counts the days a series was actually done).
//
// READ access (CalendarIDsFor), unlike marking a day: statistics shows a
// shared calendar's series to everyone who can see it.
//
// ⚠ to_char on a DATE column does not depend on the session's zone — it is not
// a timestamptz, so the day comes out as it was stored.
func ListOccurrences(ctx context.Context, userID string, from, to time.Time) ([]OccurrenceRow, error) {
	if to.Before(from) {
		return nil, ErrInvalidRange
	}
	if to.Sub(from) > 366*24*time.Hour {
		return nil, ErrRangeTooLarge
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT o.task_id::text, to_char(o.occurrence, 'YYYY-MM-DD'), o.state
		  FROM task_occurrence o
		  JOIN task t ON t.id = o.task_id
		 WHERE t.calendar_id = ANY($1) AND o.occurrence BETWEEN $2 AND $3
		 ORDER BY o.occurrence`,
		calIDs, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OccurrenceRow{}
	for rows.Next() {
		var r OccurrenceRow
		if err := rows.Scan(&r.TaskID, &r.Occurrence, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
