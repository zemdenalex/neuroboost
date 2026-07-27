package events

import "time"

// OccurrencesInRange returns the start times of every occurrence of ev that
// falls inside the half-open range [from, to). A non-recurring event yields
// its own start, or nothing when that start is outside the range.
//
// This exists because expandRecurrence is unexported and returns []Event with
// synthetic instance IDs; callers outside this package (the reminder worker)
// want start times, not events.
//
// The nil/empty-rrule guard is not optional: expandRecurrence dereferences
// *event.Rrule, so calling it for a one-off event panics. Every handler in
// this package guards before calling it, and so must this wrapper — its
// caller is a background worker, where a panic takes down the whole process
// rather than a single request.
func OccurrencesInRange(ev Event, from, to time.Time, exceptions []time.Time) []time.Time {
	if ev.Rrule == nil || *ev.Rrule == "" {
		if ev.StartsAt.Before(from) || !ev.StartsAt.Before(to) {
			return []time.Time{}
		}
		return []time.Time{ev.StartsAt}
	}

	instances := expandRecurrence(ev, from, to, exceptions)
	starts := make([]time.Time, 0, len(instances))
	for _, inst := range instances {
		starts = append(starts, inst.StartsAt)
	}
	return starts
}
