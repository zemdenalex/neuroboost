package events

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"time"
)

// errNoSuchOccurrence means the date in a synthetic instance ID is not one the
// recurrence rule produces.
var errNoSuchOccurrence = errors.New("no such occurrence")

// occurrenceWindow returns the absolute start and end of the recurring event's
// occurrence falling on the given date.
//
// The date alone cannot reconstruct the instant: a synthetic instance ID carries
// only "YYYY-MM-DD" while the occurrence has a time of day, and across a DST
// boundary that time of day is not a fixed offset from the parent's. So the rule
// is re-expanded for that single day instead of doing date arithmetic — the same
// code path that minted the ID, which is what stops the two from drifting.
//
// Exceptions are deliberately not applied: an occurrence that has already been
// detached must still resolve, otherwise editing it a second time would 404.
func occurrenceWindow(parent Event, date time.Time) (start, end time.Time, ok bool) {
	day := date.UTC().Format(instanceIDDateLayout)
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	duration := parent.EndsAt.Sub(parent.StartsAt)
	for _, s := range OccurrencesInRange(parent, dayStart, dayStart.AddDate(0, 0, 1), nil) {
		// The window filter also admits an occurrence that began the previous day
		// and spills into this one; match on the ID's own date basis instead.
		if s.UTC().Format(instanceIDDateLayout) == day {
			return s.UTC(), s.UTC().Add(duration), true
		}
	}

	return time.Time{}, time.Time{}, false
}

// loadOccurrence fetches the parent series and resolves the occurrence the
// synthetic ID names. Both failures are a 404 to the caller: a parent that is
// gone and a date the rule never produced are equally "no such event".
func loadOccurrence(ctx context.Context, userID, parentID string, date time.Time) (parent *Event, start, end time.Time, err error) {
	parent, err = getEvent(ctx, userID, parentID)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	start, end, ok := occurrenceWindow(*parent, date)
	if !ok {
		return nil, time.Time{}, time.Time{}, errNoSuchOccurrence
	}

	return parent, start, end, nil
}

// applySeriesDelta translates a time change made on one occurrence into the
// equivalent change on the parent row.
//
// Writing the new times absolutely would re-anchor the whole series to the
// edited occurrence's date — dragging the third Wednesday half an hour later
// would move the series start to that Wednesday. "All events" means "shift every
// occurrence by the same amount", so the parent moves by the delta.
//
// A nil bound was not part of the request and is left exactly as it was.
func applySeriesDelta(parentStart, parentEnd, occStart, occEnd time.Time, newStart, newEnd *time.Time) (time.Time, time.Time) {
	start, end := parentStart, parentEnd
	if newStart != nil {
		start = parentStart.Add(newStart.Sub(occStart))
	}
	if newEnd != nil {
		end = parentEnd.Add(newEnd.Sub(occEnd))
	}
	return start, end
}

// mergeOccurrence builds the detached event that will stand in for one
// occurrence: the parent's content, the occurrence's own times, and whatever the
// request changes on top.
//
// The recurrence rule is dropped and can never be set from the request — a
// detached occurrence that carried a rule would quietly become a second series.
func mergeOccurrence(parent Event, occStart, occEnd time.Time, req UpdateEventRequest) Event {
	e := parent
	e.ID = ""
	e.Rrule = nil
	e.RecurringEventID = nil
	e.StartsAt = occStart
	e.EndsAt = occEnd

	if req.Title != nil {
		e.Title = *req.Title
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.AllDay != nil {
		e.AllDay = *req.AllDay
	}
	if req.Timezone != nil {
		e.Timezone = *req.Timezone
	}
	if req.Location != nil {
		e.Location = req.Location
	}
	if req.Color != nil {
		e.Color = req.Color
	}
	if req.Tags != nil {
		e.Tags = req.Tags
	}
	// Copied from the parent unless the request says otherwise. Falling through
	// to the user's default preset here would silently change the reminders on an
	// occurrence that was only moved.
	if req.ReminderOffsets != nil {
		e.ReminderOffsets = *req.ReminderOffsets
	}
	if req.IsWorkEvent != nil {
		e.IsWorkEvent = *req.IsWorkEvent
	}

	return e
}

// detachOccurrence writes the standalone replacement event and marks the
// occurrence skipped on the series, in a single transaction.
//
// The transaction is load-bearing: these are two writes, and a failure on the
// second would leave an orphan event alongside a still-visible original — the
// exact duplicate this feature exists to prevent, and one a happy-path test
// never shows.
//
// listEvents picks the replacement up on its own (it has no rrule and overlaps
// the window), so event_exception.replacement_event_id is bookkeeping that lets
// a later "all events" edit find what was detached.
func detachOccurrence(ctx context.Context, userID string, e Event, parentID string, occurrence time.Time) (*Event, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// The replacement stays on the series' own calendar, not the caller's
	// personal one — otherwise detaching an occurrence from a shared calendar
	// would move it into the editor's private calendar and it would vanish
	// for everyone else who could see the series.
	var calendarID string
	if err := tx.QueryRow(ctx, `SELECT calendar_id FROM event WHERE id = $1`, parentID).Scan(&calendarID); err != nil {
		return nil, err
	}

	// 🔴 Detaching is a WRITE to somebody else's series, and until 2026-08-16
	// nothing on this path checked for more than read access. loadOccurrence
	// reaches the parent through getEvent, which scopes by CalendarIDsFor - every
	// calendar the caller may READ. So a viewer on a shared calendar could
	// PATCH .../{id}:{date}?scope=occurrence and both hide the real occurrence
	// (via the event_exception below) and substitute a row of their own. To the
	// other members that looks like the owner moved the meeting.
	//
	// Latent only until invitations create the first viewer membership.
	if _, err := calendars.WritableIDFor(ctx, userID, calendarID); err != nil {
		return nil, err
	}

	// Whatever this occurrence was already detached into, if anything.
	//
	// 🔴 Read BEFORE inserting the new replacement, and inside the same
	// transaction. The upsert below repoints the exception at the new row, which
	// leaves the previous replacement with nothing pointing at it — but it stays
	// in `event`, has no rrule, and therefore keeps rendering. Migration 000013
	// stops the second member creating a second EXCEPTION; only this stops them
	// creating a second visible EVENT. The test asserts both counts separately
	// for exactly that reason: the row count was already right while the
	// calendar still showed two copies.
	//
	// FOR UPDATE also serialises two members detaching the same occurrence at
	// once. When no row exists yet it locks nothing, and the unique constraint
	// makes the loser fail rather than duplicate.
	var previousReplacement *string
	if err := tx.QueryRow(ctx,
		`SELECT replacement_event_id::text FROM event_exception
		  WHERE event_id = $1 AND occurrence = $2 FOR UPDATE`,
		parentID, occurrence).Scan(&previousReplacement); err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	var created Event
	var tags []string
	err = tx.QueryRow(ctx, `
		INSERT INTO event (user_id, calendar_id, title, description, starts_at, ends_at, all_day, timezone,
		                   location, color, tags, task_id, is_work_event, reminder_offsets)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, userID, calendarID, e.Title, e.Description, e.StartsAt, e.EndsAt, e.AllDay, e.Timezone,
		e.Location, e.Color, e.Tags, e.TaskID, e.IsWorkEvent, e.ReminderOffsets).Scan(
		&created.ID, &created.UserID, &created.Title, &created.Description,
		&created.StartsAt, &created.EndsAt, &created.AllDay, &created.Rrule,
		&created.Timezone, &created.Location, &created.Color, &tags,
		&created.TaskID, &created.IsWorkEvent, &created.CreatedAt, &created.UpdatedAt,
		&created.ReminderOffsets,
	)
	if err != nil {
		return nil, err
	}
	created.Tags = tags

	// ON CONFLICT: re-editing an occurrence that is already detached must point
	// the exception at the new replacement rather than fail on the unique key.
	//
	// 🔴 The conflict target is (event_id, occurrence) — WITHOUT user_id — and
	// migration 000013 makes the constraint match. An exception is shared state
	// of the series: fetchExceptions reads it by calendar and ignores user_id on
	// purpose, because one member moving next Tuesday moves it for everyone who
	// can see the calendar. While the key included user_id, a second writing
	// member detaching the same occurrence did NOT conflict, so the calendar
	// grew a second replacement event while the original was hidden once.
	//
	// user_id is still written: it records who made the exception. It just does
	// not decide whether the exception is the same one.
	_, err = tx.Exec(ctx, `
		INSERT INTO event_exception (event_id, user_id, occurrence, skipped, replacement_event_id)
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (event_id, occurrence)
		DO UPDATE SET skipped = true,
		              user_id = EXCLUDED.user_id,
		              replacement_event_id = EXCLUDED.replacement_event_id
	`, parentID, userID, occurrence, created.ID)
	if err != nil {
		return nil, err
	}

	// Remove the copy this detach supersedes. Guarded against deleting the row
	// just created, which would happen if the same replacement were somehow
	// reused.
	if previousReplacement != nil && *previousReplacement != created.ID {
		if _, err := tx.Exec(ctx, `DELETE FROM event WHERE id = $1`, *previousReplacement); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &created, nil
}
