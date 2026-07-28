package events

import (
	"context"
	"errors"
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

	var created Event
	var tags []string
	err = tx.QueryRow(ctx, `
		INSERT INTO event (user_id, title, description, starts_at, ends_at, all_day, timezone,
		                   location, color, tags, task_id, is_work_event, reminder_offsets)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, userID, e.Title, e.Description, e.StartsAt, e.EndsAt, e.AllDay, e.Timezone,
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
	_, err = tx.Exec(ctx, `
		INSERT INTO event_exception (event_id, user_id, occurrence, skipped, replacement_event_id)
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, event_id, occurrence)
		DO UPDATE SET skipped = true, replacement_event_id = EXCLUDED.replacement_event_id
	`, parentID, userID, occurrence, created.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &created, nil
}
