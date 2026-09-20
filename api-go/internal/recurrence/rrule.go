// Package recurrence holds what "it repeats" means, for everything that repeats.
//
// 🔴 A leaf: it imports nothing of ours, and events, tasks and reminders all
// import it. The parser used to live inside `events`, unexported. Tasks are
// about to need the same grammar (v0.4.11.4), and converting a task into an
// event needs both packages to know about each other — so `tasks` importing
// `events` would become an import cycle exactly when it is hardest to undo.
// The same shape `usersettings` already has for the same reason.
//
// The grammar is deliberately narrow: FREQ=DAILY|WEEKLY|MONTHLY plus
// INTERVAL/COUNT/UNTIL, and nothing else. Anything wider must be REJECTED
// rather than ignored — an RRULE that parses and never fires is the defect that
// made «каждый год» silently produce a one-off event.
package recurrence

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Rule is a parsed RRULE.
type Rule struct {
	Freq     string     // DAILY, WEEKLY, MONTHLY
	Interval int        // default 1
	Count    *int       // optional max occurrences
	Until    *time.Time // optional last day, inclusive
}

// Parse reads an RRULE such as "FREQ=DAILY;COUNT=10" or
// "FREQ=WEEKLY;UNTIL=2026-06-01;INTERVAL=2".
//
// Moved verbatim from events.parseRRule rather than rewritten: a second opinion
// about what a repeat means is a second opinion that will eventually disagree
// with the first.
func Parse(rrule string) (*Rule, error) {
	rule := &Rule{Interval: 1}

	for _, part := range strings.Split(rrule, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("malformed RRULE part: %s", part)
		}

		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])

		switch key {
		case "FREQ":
			freq := strings.ToUpper(val)
			if freq != "DAILY" && freq != "WEEKLY" && freq != "MONTHLY" {
				return nil, fmt.Errorf("unsupported FREQ: %s", val)
			}
			rule.Freq = freq

		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid INTERVAL: %s", val)
			}
			rule.Interval = n

		case "COUNT":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid COUNT: %s", val)
			}
			rule.Count = &n

		case "UNTIL":
			t, err := time.Parse("2006-01-02", val)
			if err != nil {
				return nil, fmt.Errorf("invalid UNTIL date: %s", val)
			}
			rule.Until = &t

		default:
			return nil, fmt.Errorf("unknown RRULE key: %s", key)
		}
	}

	if rule.Freq == "" {
		return nil, fmt.Errorf("RRULE missing required FREQ")
	}

	return rule, nil
}

// dayOf reduces a timestamp to the calendar DATE it shows in its own location,
// and then forgets the location.
//
// 🔴 It used to keep it — time.Date(y, m, d, …, t.Location()) — which made two
// values for the same date unequal whenever they arrived from different zones.
// And they always do: the anchor is a DATE column, read back as midnight UTC,
// while «сегодня» is LocalDay(), midnight in the user's zone. In Moscow the
// second is three hours before the first, so the anchor day was «before the
// series» and every later day counted one step short (a Monday task on
// Tuesdays). West of Greenwich the same error ran the other way.
//
// Putting every date on the UTC axis makes the arithmetic below pure date
// arithmetic: no zone to disagree about, and no 23- or 25-hour day at a
// daylight-saving change for Hours()/24 to stumble over.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Occurs reports whether `day` belongs to the series anchored at `anchor`.
//
// 🔴 Both are reduced to a calendar DAY before comparison, in whatever location
// they carry. A task has no clock — «выпить таблетки 19 сентября» is a day — and
// comparing instants would make an occurrence appear or vanish depending on the
// hour the question is asked. Callers pass times already in the user's zone;
// this function does not guess one.
func Occurs(r *Rule, anchor, day time.Time) bool {
	if r == nil {
		return false
	}
	a, d := dayOf(anchor), dayOf(day)
	if d.Before(a) {
		return false
	}

	n, ok := stepsBetween(r, a, d)
	if !ok {
		return false
	}
	if r.Count != nil && n >= *r.Count {
		return false
	}
	if r.Until != nil && d.After(dayOf(*r.Until)) {
		return false
	}
	return true
}

// stepsBetween returns which occurrence `d` is (0 for the anchor itself) and
// whether it lands on one at all.
func stepsBetween(r *Rule, a, d time.Time) (int, bool) {
	switch r.Freq {
	case "DAILY":
		days := int(d.Sub(a).Hours() / 24)
		if days%r.Interval != 0 {
			return 0, false
		}
		return days / r.Interval, true

	case "WEEKLY":
		days := int(d.Sub(a).Hours() / 24)
		if days%7 != 0 {
			return 0, false
		}
		weeks := days / 7
		if weeks%r.Interval != 0 {
			return 0, false
		}
		return weeks / r.Interval, true

	case "MONTHLY":
		// ⚠ The day of the month must match exactly: the 31st simply does not
		// occur in a 30-day month. Sliding it to the 30th would invent an
		// occurrence the user never asked for, and «раз в месяц 31-го» is a
		// thing people write knowing some months are skipped.
		if d.Day() != a.Day() {
			return 0, false
		}
		months := (d.Year()-a.Year())*12 + int(d.Month()) - int(a.Month())
		if months < 0 || months%r.Interval != 0 {
			return 0, false
		}
		return months / r.Interval, true
	}
	return 0, false
}

// Next returns the first occurrence strictly after `after`, and whether the
// series has one at all — a COUNT or UNTIL series eventually does not.
func Next(r *Rule, anchor, after time.Time) (time.Time, bool) {
	if r == nil {
		return time.Time{}, false
	}
	day := dayOf(after).AddDate(0, 0, 1)
	// A year of daily steps is 365 checks; a bounded walk is simpler to read
	// than closed-form arithmetic for three frequencies and never wrong.
	limit := dayOf(after).AddDate(3, 0, 0)
	for !day.After(limit) {
		if Occurs(r, anchor, day) {
			return day, true
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, false
}
