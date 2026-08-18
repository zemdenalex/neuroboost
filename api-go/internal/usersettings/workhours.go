package usersettings

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// Work hours: how long the user's week actually is.
//
// 🔴 Until 19.08 nothing in this product read work_start, work_end or work_days.
// The web app had a Settings section that wrote them, AuthContext copied them
// into localStorage, and no reader existed anywhere — not in the API, not in
// the frontend, not in the bot. `grep -rn work_start` found the writers and the
// type declaration and stopped there.
//
// The one place that wanted the number had hardcoded it: planning's
// AvailableHours was the literal 40, with the comment "8h × 5 days". So the
// control was decorative in every client, and the bot's new work-hours screen
// would have been decorative too — a button that saves a value nobody asks for
// is the same dead end as a button that opens a link.

// DefaultWorkDays is Monday to Friday, matching the web's own default
// (lib/settings/workDays.ts) rather than a second opinion about the work week.
var DefaultWorkDays = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

const (
	defaultDayStartMin = 9 * 60  // 09:00
	defaultDayEndMin   = 17 * 60 // 17:00
)

// WorkWeek is what the settings blob says about the shape of a week.
type WorkWeek struct {
	StartMinutes int // minutes past midnight, local to the user
	EndMinutes   int
	Days         []string
}

// Hours is the length of the working week, in hours.
//
// Zero when the day is empty or inverted rather than a negative number: an
// inverted range is a data error, and the honest answer to "how many hours do
// you have" is none, not minus six.
func (w WorkWeek) Hours() float64 {
	perDay := w.EndMinutes - w.StartMinutes
	if perDay <= 0 || len(w.Days) == 0 {
		return 0
	}
	return float64(perDay) * float64(len(w.Days)) / 60
}

// ParseWorkWeek reads the blob, falling back per FIELD rather than per blob.
//
// A user who set only their end time keeps the default start, instead of having
// the whole section ignored because one key was missing — which is how a
// partially-filled profile silently reverts to 09:00–17:00 and looks like the
// save failed.
func ParseWorkWeek(raw []byte) WorkWeek {
	w := WorkWeek{
		StartMinutes: defaultDayStartMin,
		EndMinutes:   defaultDayEndMin,
		Days:         append([]string(nil), DefaultWorkDays...),
	}
	if len(raw) == 0 {
		return w
	}

	var blob struct {
		WorkStart *string  `json:"work_start"`
		WorkEnd   *string  `json:"work_end"`
		WorkDays  []string `json:"work_days"`
	}
	if err := json.Unmarshal(raw, &blob); err != nil {
		return w
	}

	if m, ok := parseHHMM(blob.WorkStart); ok {
		w.StartMinutes = m
	}
	if m, ok := parseHHMM(blob.WorkEnd); ok {
		w.EndMinutes = m
	}
	// An explicitly empty array means "no working days" and is kept, because a
	// user who unticked every day meant it. Only a MISSING key falls back.
	if blob.WorkDays != nil {
		w.Days = blob.WorkDays
	}
	return w
}

// parseHHMM accepts "09:00" and "9:00", and rejects everything else.
//
// Written by hand rather than with time.Parse because the value is a wall-clock
// string with no date and no zone, and time.Parse would attach 1 January year 0
// in UTC — a Time that invites arithmetic no caller should be doing.
func parseHHMM(v *string) (int, bool) {
	if v == nil {
		return 0, false
	}
	s := strings.TrimSpace(*v)
	hh, mm, found := strings.Cut(s, ":")
	if !found {
		return 0, false
	}
	h, err1 := strconv.Atoi(hh)
	m, err2 := strconv.Atoi(mm)
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// LoadWorkWeek reads one user's work week, defaulting on any failure — the same
// contract as LoadReminders, and for the same reason: a planning page that
// errors because a settings blob is unreadable is worse than one showing 40.
func LoadWorkWeek(ctx context.Context, userID string) WorkWeek {
	if db == nil {
		return ParseWorkWeek(nil)
	}
	var raw []byte
	if err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(settings, '{}') FROM "user" WHERE id = $1`, userID).Scan(&raw); err != nil {
		return ParseWorkWeek(nil)
	}
	return ParseWorkWeek(raw)
}
