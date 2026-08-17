package reminders

import (
	"time"

	"neuroboost/api-go/internal/usersettings"
)

// quietGraceMinutes: a reminder with this much notice or less is dropped
// rather than shifted. Moving a "15 minutes before" reminder to the end of
// quiet hours would deliver it after the thing it warns about.
const quietGraceMinutes = 15

// ShiftForQuietHours reports when a reminder should actually be delivered.
// The second return value is false when the reminder should be dropped.
func ShiftForQuietHours(remindAt time.Time, minutesBefore int, quietStart, quietEnd string, loc *time.Location) (time.Time, bool) {
	startMin, okStart := usersettings.ParseHHMM(quietStart)
	endMin, okEnd := usersettings.ParseHHMM(quietEnd)
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
// It returns both the local day the digest covers and the exact instant it is
// due. 🔴 The instant used to be computed here and thrown away, and the row was
// written with remind_at = NOW(); because the scan window looks AHEAD, the
// digest went out up to a minute early — 08:00 arrived at 07:59:12. Returning
// fireAt lets the row carry its real time, and PendingHandler's
// `remind_at <= NOW()` then holds it until the minute it was asked for.
func DigestDue(windowStart, windowEnd time.Time, digestAt string, loc *time.Location) (day time.Time, fireAt time.Time, ok bool) {
	atMin, parsed := usersettings.ParseHHMM(digestAt)
	if !parsed {
		return time.Time{}, time.Time{}, false
	}
	local := windowStart.In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	// Check today and tomorrow: a window can straddle local midnight.
	for _, d := range []time.Time{midnight, midnight.AddDate(0, 0, 1)} {
		at := d.Add(time.Duration(atMin) * time.Minute)
		if !at.Before(windowStart) && at.Before(windowEnd) {
			return d, at, true
		}
	}
	return time.Time{}, time.Time{}, false
}
