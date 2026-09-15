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

	// Repeat is an RFC 5545 RRULE, or empty when there is none.
	// RepeatAsked is the state an API request cannot express: the word
	// "повтор" was written but no frequency was given, so the card must ask.
	Repeat      string
	RepeatAsked bool

	AllDay bool

	// IsTask means the word "задача" was used: the bot creates a task and an
	// event bound to it, so the entry can be ticked off.
	IsTask bool

	// Colour is a palette name from web/src/lib/calendar/palette.ts. Anything
	// else would be stored and then not painted.
	Colour string

	// Calendar is the word that named a calendar; resolving it to an id needs
	// the user's calendar list and therefore happens outside the parser.
	Calendar string

	Tags []string
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
