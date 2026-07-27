package events

import "time"

// OccurrencesInRange returns the start times of every occurrence of ev that
// falls inside [from, to). For a non-recurring event that is either its own
// start or nothing.
//
// This exists because expandRecurrence is unexported and returns []Event with
// synthetic instance IDs; callers outside this package (the reminder worker)
// want start times, not events.
func OccurrencesInRange(ev Event, from, to time.Time, exceptions []time.Time) []time.Time {
	instances := expandRecurrence(ev, from, to, exceptions)
	starts := make([]time.Time, 0, len(instances))
	for _, inst := range instances {
		starts = append(starts, inst.StartsAt)
	}
	return starts
}
