// Package remindrule is «remind me regularly until the deadline» (Лешик and
// Denis, 07.11): one rule on a task or an event, expanded by the reminder
// scanner into ordinary journal rows.
//
// A leaf package on purpose: tasks, events and reminders all need it, and
// reminders already imports events.
package remindrule

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Rule is exactly one of two shapes:
//
//	{"every_minutes": 5}                          — every N minutes
//	{"every_days": 1, "times": ["12:00","16:00"]} — every K days at these times
//
// Both run from now until the deadline (a task's due, an event's start),
// the deadline day included up to the deadline moment.
type Rule struct {
	EveryMinutes int      `json:"every_minutes,omitempty"`
	EveryDays    int      `json:"every_days,omitempty"`
	Times        []string `json:"times,omitempty"`
}

const (
	// MinMinutes matches the smallest nag interval: below it the bot is a
	// stream, not a reminder.
	MinMinutes = 5
	MaxMinutes = 7 * 24 * 60
	MaxDays    = 30
	MaxTimes   = 8
)

var ErrInvalid = errors.New("invalid remind rule")

// Validate also normalises the times (sorted, deduplicated, "9:00" → "09:00").
func (r *Rule) Validate() error {
	interval := r.EveryMinutes != 0
	daily := r.EveryDays != 0 || len(r.Times) > 0
	switch {
	case interval == daily:
		return fmt.Errorf("%w: give every_minutes, or every_days with times", ErrInvalid)
	case interval:
		if r.EveryMinutes < MinMinutes || r.EveryMinutes > MaxMinutes {
			return fmt.Errorf("%w: every_minutes must be %d–%d", ErrInvalid, MinMinutes, MaxMinutes)
		}
		return nil
	}
	if r.EveryDays < 1 || r.EveryDays > MaxDays {
		return fmt.Errorf("%w: every_days must be 1–%d", ErrInvalid, MaxDays)
	}
	if len(r.Times) == 0 || len(r.Times) > MaxTimes {
		return fmt.Errorf("%w: 1–%d times", ErrInvalid, MaxTimes)
	}
	seen := map[int]bool{}
	mins := []int{}
	for _, s := range r.Times {
		m, ok := parseHHMM(s)
		if !ok {
			return fmt.Errorf("%w: time %q is not HH:MM", ErrInvalid, s)
		}
		if !seen[m] {
			seen[m] = true
			mins = append(mins, m)
		}
	}
	sort.Ints(mins)
	r.Times = r.Times[:0]
	for _, m := range mins {
		r.Times = append(r.Times, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	return nil
}

func parseHHMM(s string) (int, bool) {
	var h, m int
	if n, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || n != 2 {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// TaskDeadline is the moment a task's rule runs up to. A task due at local
// midnight is a date-only deadline (the bot and the web both store a bare date
// that way), and «до срока» then means through that whole day: 23:59, not
// 00:00 — otherwise «каждый день в 12» would skip the deadline day itself,
// the one day that matters most (Denis, 07.11: «Yes, up to the due moment»).
func TaskDeadline(due time.Time, loc *time.Location) time.Time {
	l := due.In(loc)
	if l.Hour() == 0 && l.Minute() == 0 && l.Second() == 0 {
		return time.Date(l.Year(), l.Month(), l.Day(), 23, 59, 0, 0, loc)
	}
	return due
}

// AllDayDeadline is the end of an all-day event's day. The bot stores the
// day as local midnight; a value that is not local midnight (written as a UTC
// midnight) is read by its UTC date.
func AllDayDeadline(start time.Time, loc *time.Location) time.Time {
	l := start.In(loc)
	if l.Hour() == 0 && l.Minute() == 0 {
		return time.Date(l.Year(), l.Month(), l.Day(), 23, 59, 0, 0, loc)
	}
	u := start.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 23, 59, 0, 0, loc)
}

// Expand returns the moments the rule fires inside the half-open window
// [from, to), strictly before the deadline, oldest first.
//
// It is arithmetic, not a walk from «now» to the deadline: every 5 minutes for
// a month is 8640 moments, and the scanner asks once a minute.
func (r Rule) Expand(deadline time.Time, loc *time.Location, from, to time.Time) []time.Time {
	out := []time.Time{}
	if !from.Before(to) || !from.Before(deadline) {
		return out
	}
	if r.EveryMinutes > 0 {
		// Anchored on the deadline: deadline − k·N for k ≥ 1. The anchor makes
		// every moment the same on every scan, so the journal's unique index
		// (occurrence, minutes_before) dedupes overlapping windows.
		step := time.Duration(r.EveryMinutes) * time.Minute
		// deadline − to < k·N ≤ deadline − from
		kMin := int64(deadline.Sub(to)/step) + 1
		if kMin < 1 {
			kMin = 1
		}
		kMax := int64(deadline.Sub(from) / step)
		for k := kMax; k >= kMin; k-- {
			t := deadline.Add(-time.Duration(k) * step)
			if !t.Before(from) && t.Before(to) {
				out = append(out, t)
			}
		}
		return out
	}

	// Every K days, counted back from the deadline's own day, so the deadline
	// day is always one of them.
	dl := deadline.In(loc)
	deadlineDay := time.Date(dl.Year(), dl.Month(), dl.Day(), 0, 0, 0, 0, loc)
	f := from.In(loc)
	day := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, loc)
	for !day.After(deadlineDay) && day.Before(to) {
		if daysBetween(day, deadlineDay)%r.EveryDays == 0 {
			for _, s := range r.Times {
				m, ok := parseHHMM(s)
				if !ok {
					continue
				}
				t := time.Date(day.Year(), day.Month(), day.Day(), m/60, m%60, 0, 0, loc)
				if !t.Before(from) && t.Before(to) && t.Before(deadline) {
					out = append(out, t)
				}
			}
		}
		day = day.AddDate(0, 0, 1)
	}
	return out
}

// daysBetween counts calendar days, not 24-hour blocks: across a DST change a
// day is 23 or 25 hours long and division by 24h would miscount.
func daysBetween(a, b time.Time) int {
	ua := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	ub := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}
