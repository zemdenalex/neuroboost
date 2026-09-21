// Package statgrid turns events, tasks and reflections into the numbers the
// statistics screen draws (spec 22.09). Pure: no Telegram, no API, no clock —
// every «day» and «hour» arrives already placed in the user's zone.
package statgrid

import (
	"sort"
	"time"
)

// Span is one booked interval.
type Span struct{ Start, End time.Time }

func clip(s Span, from, to time.Time) (Span, bool) {
	if s.Start.Before(from) {
		s.Start = from
	}
	if s.End.After(to) {
		s.End = to
	}
	return s, s.End.After(s.Start)
}

// Busy is the time covered by at least one span inside [from, to).
//
// 🔴 A union, not a sum — Denis 20.09: «не считал дважды события в одно и то
// же время». Planned is the sum; the gap between them is the overlap.
func Busy(spans []Span, from, to time.Time) time.Duration {
	var in []Span
	for _, s := range spans {
		if c, ok := clip(s, from, to); ok {
			in = append(in, c)
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].Start.Before(in[j].Start) })
	var total time.Duration
	var cur Span
	for i, s := range in {
		if i == 0 {
			cur = s
			continue
		}
		if !s.Start.After(cur.End) {
			if s.End.After(cur.End) {
				cur.End = s.End
			}
			continue
		}
		total += cur.End.Sub(cur.Start)
		cur = s
	}
	if len(in) > 0 {
		total += cur.End.Sub(cur.Start)
	}
	return total
}

// Planned is the plain sum of the spans' time inside [from, to).
func Planned(spans []Span, from, to time.Time) time.Duration {
	var total time.Duration
	for _, s := range spans {
		if c, ok := clip(s, from, to); ok {
			total += c.End.Sub(c.Start)
		}
	}
	return total
}
