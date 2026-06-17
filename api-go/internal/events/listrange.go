package events

import "time"

// parseListRange resolves the start/end query params for ListHandler. If either is
// empty it defaults to the week containing `now` (matching the prior behaviour).
// Otherwise both are parsed as RFC3339 and the window is validated as start < end.
// On failure it returns an (errorCode, message) pair to respond 400 with; on success
// both are "". Pure so the parsing/validation can be tested without a database.
func parseListRange(startParam, endParam string, now time.Time) (time.Time, time.Time, string, string) {
	if startParam == "" || endParam == "" {
		weekStart := now.AddDate(0, 0, -int(now.Weekday()))
		weekEnd := weekStart.AddDate(0, 0, 7)
		return weekStart, weekEnd, "", ""
	}

	start, err := time.Parse(time.RFC3339, startParam)
	if err != nil {
		return time.Time{}, time.Time{}, "INVALID_START", "Invalid start date format"
	}

	end, err := time.Parse(time.RFC3339, endParam)
	if err != nil {
		return time.Time{}, time.Time{}, "INVALID_END", "Invalid end date format"
	}

	if err := validateTimeRange(start, end); err != nil {
		return time.Time{}, time.Time{}, "INVALID_RANGE", "Start must be before end"
	}

	return start, end, "", ""
}
