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
	if state != StateDone && state != StateSkipped {
		return fmt.Errorf("unknown occurrence state: %s", state)
	}

	rule, anchor, _, err := repeatOf(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if !recurrence.Occurs(rule, anchor, day) {
		return ErrNotAnOccurrence
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO task_occurrence (user_id, task_id, occurrence, state)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, occurrence)
		DO UPDATE SET state = EXCLUDED.state, acted_at = NOW()`,
		userID, taskID, day.Format("2006-01-02"), state)
	return err
}

// MarkOccurrenceAt records a day from an instant, resolving the day where the
// user is. This is what a button press should call: the caller knows "now", not
// "which calendar day the user thinks it is".
func MarkOccurrenceAt(ctx context.Context, userID, taskID string, at time.Time, state string) error {
	_, _, tz, err := repeatOf(ctx, userID, taskID)
	if err != nil {
		return err
	}
	return MarkOccurrence(ctx, userID, taskID, LocalDay(at, tz), state)
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
