package events

import (
	"context"

	"neuroboost/api-go/internal/calendars"
)

// decorateSharing fills IsShared and AuthorName on events already loaded and
// scoped. It is applied after scanning rather than folded into each SELECT on
// purpose: this package scans the same column list in eight places, and two more
// columns in each is an invitation for one of them to drift. One decorator, four
// call sites, and a handler that forgets it produces events with no badge —
// visibly wrong — rather than a scan that misaligns its arguments.
//
// Two queries regardless of how many events came back.
func decorateSharing(ctx context.Context, viewerID string, events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	calendarIDs := make([]string, 0, len(events))
	authorIDs := make([]string, 0, len(events))
	seenCalendar := map[string]bool{}
	seenAuthor := map[string]bool{}
	for _, e := range events {
		if e.CalendarID != "" && !seenCalendar[e.CalendarID] {
			seenCalendar[e.CalendarID] = true
			calendarIDs = append(calendarIDs, e.CalendarID)
		}
		// Only strangers are looked up. The caller's own name is never rendered,
		// so fetching it would be a row read to throw away.
		if e.UserID != "" && e.UserID != viewerID && !seenAuthor[e.UserID] {
			seenAuthor[e.UserID] = true
			authorIDs = append(authorIDs, e.UserID)
		}
	}

	shared, err := calendars.SharedIDs(ctx, calendarIDs)
	if err != nil {
		return err
	}
	names, err := calendars.DisplayNames(ctx, authorIDs)
	if err != nil {
		return err
	}

	for _, e := range events {
		e.IsShared = shared[e.CalendarID]

		// 🔴 The author is named only inside a shared calendar. A personal
		// calendar holds exactly one member, so a foreign user_id there means
		// history — an import, a repaired row, an account merge — not a person
		// the viewer could point at. Naming them would invent a collaborator.
		if !e.IsShared {
			e.AuthorName = nil
			continue
		}
		if name, ok := names[e.UserID]; ok {
			label := name
			e.AuthorName = &label
		} else {
			e.AuthorName = nil
		}
	}
	return nil
}

// decorateSharingSlice is the same for a slice of values rather than pointers.
func decorateSharingSlice(ctx context.Context, viewerID string, events []Event) error {
	pointers := make([]*Event, len(events))
	for i := range events {
		pointers[i] = &events[i]
	}
	return decorateSharing(ctx, viewerID, pointers)
}
