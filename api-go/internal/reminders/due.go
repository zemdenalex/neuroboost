// Package reminders computes which notifications are due and hands them to
// the notifier. The time math lives in pure functions (due.go, quiet.go) so it
// can be tested without a database; scan.go and worker.go hold the I/O.
package reminders

import "time"

// Due is one reminder that ought to exist: a single occurrence paired with a
// single offset. It maps 1:1 onto a row in the reminder journal.
type Due struct {
	OccurrenceStart time.Time
	MinutesBefore   int
	RemindAt        time.Time
}

// DueReminders returns the (occurrence, offset) pairs whose remind-at moment
// falls inside the half-open window [windowStart, windowEnd).
//
// Half-open matters: the worker scans overlapping windows so a skipped tick
// (deploy, restart) cannot lose a reminder. The unique index makes the overlap
// safe, but a closed window would make every boundary reminder collide on
// purpose, which would mask genuine duplicate bugs behind ON CONFLICT.
//
// The caller supplies occurrences rather than an event, which keeps this
// function independent of RRULE parsing and of the database.
func DueReminders(occurrences []time.Time, offsets []int, windowStart, windowEnd time.Time) []Due {
	due := []Due{}
	for _, occ := range occurrences {
		for _, off := range offsets {
			// Negative offsets are not "after the event": -1 is the snooze
			// sentinel and -2 the digest sentinel stored in
			// reminder.minutes_before. Neither may be produced by a scan.
			if off < 0 {
				continue
			}
			remindAt := occ.Add(-time.Duration(off) * time.Minute)
			if remindAt.Before(windowStart) || !remindAt.Before(windowEnd) {
				continue
			}
			due = append(due, Due{
				OccurrenceStart: occ,
				MinutesBefore:   off,
				RemindAt:        remindAt,
			})
		}
	}
	return due
}
