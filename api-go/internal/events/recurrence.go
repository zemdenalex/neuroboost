package events

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RecurrenceRule represents a parsed RRULE for event recurrence
type RecurrenceRule struct {
	Freq     string     // DAILY, WEEKLY, MONTHLY
	Interval int        // default 1
	Count    *int       // optional max occurrences
	Until    *time.Time // optional end date
}

// parseRRule parses an RRULE string like "FREQ=DAILY;COUNT=10" or "FREQ=WEEKLY;UNTIL=2026-06-01;INTERVAL=2"
func parseRRule(rrule string) (*RecurrenceRule, error) {
	rule := &RecurrenceRule{Interval: 1}

	parts := strings.Split(rrule, ";")
	for _, part := range parts {
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

// expandRecurrence generates event instances from a recurring event within the given time range.
// The parent event itself is NOT included — only expanded occurrences.
func expandRecurrence(event Event, rangeStart, rangeEnd time.Time, exceptions []time.Time) []Event {
	rule, err := parseRRule(*event.Rrule)
	if err != nil {
		return nil
	}

	duration := event.EndsAt.Sub(event.StartsAt)
	var instances []Event
	count := 0     // occurrence index from StartsAt — drives COUNT
	monthStep := 0 // months advanced from StartsAt, for MONTHLY day-of-month anchoring

	// maxInstances bounds the instances RETURNED for one event in one query (so a
	// huge window can't produce an unbounded payload). It is deliberately a cap on
	// emitted-in-window instances, NOT on occurrences-from-start: the latter made a
	// long-running event (e.g. a daily one viewed >366 days after its start) exhaust
	// the budget before the loop reached the window, silently dropping it.
	// maxIterations is an absolute loop ceiling — it MUST stay comfortably larger
	// than the largest realistic event-age-in-steps (100000 ≈ 273 years of daily),
	// or this exact vanishing bug returns.
	const maxInstances = 366
	const maxIterations = 100000

	occurrence := event.StartsAt
	for iter := 0; iter < maxIterations; iter++ {
		// Stop if past range end
		if occurrence.After(rangeEnd) {
			break
		}

		// Stop once past the UNTIL day. UNTIL is a DATE value (parsed at midnight)
		// and, per iCalendar, the whole UNTIL day is inclusive — so compare against
		// the start of the following day rather than the bare midnight instant.
		if rule.Until != nil && !occurrence.Before(rule.Until.AddDate(0, 0, 1)) {
			break
		}

		// Stop if COUNT reached
		if rule.Count != nil && count >= *rule.Count {
			break
		}

		// Stop once the response is full
		if len(instances) >= maxInstances {
			break
		}

		occurrenceEnd := occurrence.Add(duration)

		// Check if this occurrence is within the range
		if occurrenceEnd.After(rangeStart) && occurrence.Before(rangeEnd) {
			// Check if this occurrence is an exception (compare dates only)
			if !isException(occurrence, exceptions) {
				inst := event
				inst.StartsAt = occurrence
				inst.EndsAt = occurrenceEnd
				inst.ID = fmt.Sprintf("%s:%s", event.ID, occurrence.Format("2006-01-02"))
				parentID := event.ID
				inst.RecurringEventID = &parentID
				instances = append(instances, inst)
			}
		}

		count++

		// Advance to next occurrence
		switch rule.Freq {
		case "DAILY":
			occurrence = occurrence.AddDate(0, 0, rule.Interval)
		case "WEEKLY":
			occurrence = occurrence.AddDate(0, 0, rule.Interval*7)
		case "MONTHLY":
			// Anchor each occurrence to the original start day-of-month and SKIP
			// months that lack it (RFC 5545 §3.3.10), instead of letting AddDate
			// drift (e.g. Jan-31 + 1mo → Mar-3). Skipped months emit no instance.
			advanced := false
			for tries := 0; tries < 120; tries++ {
				monthStep += rule.Interval
				if next, ok := monthlyOccurrence(event.StartsAt, monthStep); ok {
					occurrence = next
					advanced = true
					break
				}
			}
			if !advanced {
				return instances // no valid month within the lookahead window
			}
		}
	}

	return instances
}

// daysInMonth returns the number of days in the given month, handling leap February.
func daysInMonth(year int, month time.Month) int {
	// Day 0 of the following month is the last day of this month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// monthlyOccurrence returns the instant `months` months after start, preserving the
// start's day-of-month and clock time. ok is false when the target month has no such
// day (e.g. the 31st of a 30-day month) — signaling that month should be skipped.
func monthlyOccurrence(start time.Time, months int) (time.Time, bool) {
	total := int(start.Month()) - 1 + months
	year := start.Year() + total/12
	month := time.Month(total%12 + 1)
	day := start.Day()
	if day > daysInMonth(year, month) {
		return time.Time{}, false
	}
	return time.Date(year, month, day, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), start.Location()), true
}

// isException checks if the given occurrence date matches any exception date (date-only comparison)
func isException(occurrence time.Time, exceptions []time.Time) bool {
	occDate := occurrence.Format("2006-01-02")
	for _, ex := range exceptions {
		if ex.Format("2006-01-02") == occDate {
			return true
		}
	}
	return false
}

// fetchExceptions retrieves skipped exception dates for a recurring event
func fetchExceptions(ctx context.Context, userID, eventID string) []time.Time {
	rows, err := db.Pool.Query(ctx,
		`SELECT occurrence FROM event_exception WHERE user_id = $1 AND event_id = $2 AND skipped = true`,
		userID, eventID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var exceptions []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err == nil {
			exceptions = append(exceptions, t)
		}
	}
	return exceptions
}

// buildRRule builds an RRULE string from components
func buildRRule(freq string, interval int, count *int, until *time.Time) string {
	parts := []string{fmt.Sprintf("FREQ=%s", strings.ToUpper(freq))}

	if interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", interval))
	}

	if count != nil {
		parts = append(parts, fmt.Sprintf("COUNT=%d", *count))
	}

	if until != nil {
		parts = append(parts, fmt.Sprintf("UNTIL=%s", until.Format("2006-01-02")))
	}

	return strings.Join(parts, ";")
}
