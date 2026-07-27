package reminders

import (
	"strconv"
	"strings"
	"time"
)

// quietGraceMinutes: a reminder with this much notice or less is dropped
// rather than shifted. Moving a "15 minutes before" reminder to the end of
// quiet hours would deliver it after the thing it warns about.
const quietGraceMinutes = 15

// parseHHMM turns "08:00" into minutes since local midnight. ok=false for
// anything unparseable — the settings blob is user-writable.
func parseHHMM(v string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// ShiftForQuietHours reports when a reminder should actually be delivered.
// The second return value is false when the reminder should be dropped.
func ShiftForQuietHours(remindAt time.Time, minutesBefore int, quietStart, quietEnd string, loc *time.Location) (time.Time, bool) {
	startMin, okStart := parseHHMM(quietStart)
	endMin, okEnd := parseHHMM(quietEnd)
	if !okStart || !okEnd || startMin == endMin {
		return remindAt, true // quiet hours not configured
	}

	local := remindAt.In(loc)
	nowMin := local.Hour()*60 + local.Minute()

	// The window may wrap midnight (22:00–07:00) or not (01:00–07:00).
	var inQuiet bool
	if startMin < endMin {
		inQuiet = nowMin >= startMin && nowMin < endMin
	} else {
		inQuiet = nowMin >= startMin || nowMin < endMin
	}
	if !inQuiet {
		return remindAt, true
	}
	if minutesBefore <= quietGraceMinutes {
		return time.Time{}, false
	}

	// End of the quiet window: today if we are before it, tomorrow if we are
	// in the pre-midnight half of a wrapping window.
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := day.Add(time.Duration(endMin) * time.Minute)
	if !end.After(local) {
		end = end.AddDate(0, 0, 1)
	}
	return end, true
}

// DigestDue reports whether the user's local clock crosses digestAt inside the
// half-open window [windowStart, windowEnd). The returned time is the LOCAL
// midnight of the day the digest belongs to; it goes into
// reminder.occurrence_start so the unique index dedupes per local day rather
// than per UTC day.
func DigestDue(windowStart, windowEnd time.Time, digestAt string, loc *time.Location) (time.Time, bool) {
	atMin, ok := parseHHMM(digestAt)
	if !ok {
		return time.Time{}, false
	}
	local := windowStart.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	// Check today and tomorrow: a window can straddle local midnight.
	for _, d := range []time.Time{day, day.AddDate(0, 0, 1)} {
		fireAt := d.Add(time.Duration(atMin) * time.Minute)
		if !fireAt.Before(windowStart) && fireAt.Before(windowEnd) {
			return d, true
		}
	}
	return time.Time{}, false
}
