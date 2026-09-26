package events

import (
	"context"
	"time"

	"neuroboost/api-go/internal/calendars"
)

// ListExpanded answers what GET /api/events answers: the events userID can see
// in [start, end), each repeating series expanded into its occurrences there.
//
// Exported for the planner (internal/planning), which read event rows by
// starts_at and so counted a weekly meeting only in the week of its first row.
// One reader of «what is on the calendar» is the point: a second one drifts.
func ListExpanded(ctx context.Context, userID string, start, end time.Time) ([]Event, error) {
	events, err := listEvents(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	// Read the caller's calendars ONCE, above the loop. fetchExceptions used to
	// do this itself, on every recurring event, which made half of the
	// handler's queries redundant, and, worse, swallowed the failure as "no
	// exceptions", putting deleted occurrences back on the calendar.
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// [] rather than nil: an empty week must encode as [], not null (gotcha 6).
	expanded := []Event{}
	for _, ev := range events {
		if ev.Rrule == nil || *ev.Rrule == "" {
			expanded = append(expanded, ev)
			continue
		}
		if _, perr := parseRRule(*ev.Rrule); perr != nil {
			// Invalid rrule: include the parent event as-is.
			expanded = append(expanded, ev)
			continue
		}
		exceptions, ferr := fetchExceptions(ctx, calIDs, ev.ID)
		if ferr != nil {
			// Refusing is the honest answer. Continuing with no exceptions
			// would silently redraw occurrences the user has deleted.
			return nil, ferr
		}
		expanded = append(expanded, expandRecurrence(ev, start, end, exceptions)...)
	}
	return expanded, nil
}
