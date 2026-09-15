package parse

import "time"

// Draft is what the recognisers fill in, one field each.
//
// It is deliberately NOT the API request shape. The request cannot express
// "a repeat was asked for but no frequency was given", and that state is the
// whole reason the confirmation card exists.
type Draft struct {
	// Day is local midnight of the chosen day. HasDay says whether anything
	// chose it — a zero Time would be indistinguishable from "1 January, year 1".
	Day    time.Time
	HasDay bool

	// Start and End are offsets from Day's midnight, not instants: the day and
	// the time are chosen by different recognisers and may arrive in either
	// order. HasEnd stays false when only a start was given, so the caller —
	// not the parser — decides the default length.
	Start   time.Duration
	End     time.Duration
	HasTime bool
	HasEnd  bool
}

// StartsAt composes the day and the start offset. Only meaningful when both
// HasDay and HasTime are true; the caller decides what to do otherwise.
func (d Draft) StartsAt() time.Time { return d.Day.Add(d.Start) }

// EndsAt composes the day and the end offset, rolling past midnight when the
// range does (23:00–01:00 is two hours, not a negative twenty-two).
func (d Draft) EndsAt() time.Time {
	end := d.End
	if end <= d.Start {
		end += 24 * time.Hour
	}
	return d.Day.Add(end)
}
