package events

import (
	"errors"
	"time"
)

// errInvalidTimeRange is returned when an event's end is not strictly after its start.
var errInvalidTimeRange = errors.New("end time must be after start time")

// validateTimeRange reports whether [start, end) is a valid event window:
// the end must be strictly after the start (no zero-length or inverted events).
func validateTimeRange(start, end time.Time) error {
	if !end.After(start) {
		return errInvalidTimeRange
	}
	return nil
}
