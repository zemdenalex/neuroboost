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
	count := 0
	maxExpansions := 366

	for occurrence := event.StartsAt; ; {
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

		// Safety cap
		if count >= maxExpansions {
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
			occurrence = occurrence.AddDate(0, rule.Interval, 0)
		}
	}

	return instances
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
