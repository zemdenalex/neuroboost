package daytasks

import (
	"encoding/json"
	"time"
)

// DefaultTarget is Atrioc's five, Denis's default.
const DefaultTarget = 5

// Today is midnight of the user's current day, in the user's zone.
func Today(now time.Time, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	l := now.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// ymd compares two dates by their Y-M-D alone.
func ymd(t time.Time) int { return t.Year()*10000 + int(t.Month())*100 + t.Day() }

// CanRemove is spec §5: a future day always, today until 12:00 where the user
// is, a past day never. Without the noon line, dropping an undone task at
// 23:00 would turn the day green — the colour would stop meaning anything.
func CanRemove(day, now time.Time, tz string) bool {
	today := Today(now, tz)
	switch d, t := ymd(day), ymd(today); {
	case d > t:
		return true
	case d < t:
		return false
	}
	loc := today.Location()
	return now.In(loc).Hour() < 12
}

// Target reads settings.day_tasks_target: an integer 3–7, else the default.
// Top level, not bot.*: the web reads it too, and the server needs it for the
// level.
func Target(settings []byte) int {
	var s struct {
		Target *int `json:"day_tasks_target"`
	}
	if err := json.Unmarshal(settings, &s); err != nil || s.Target == nil {
		return DefaultTarget
	}
	if *s.Target < 3 || *s.Target > 7 {
		return DefaultTarget
	}
	return *s.Target
}
