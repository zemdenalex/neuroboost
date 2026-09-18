package tasks

import (
	"testing"
	"time"
)

// 🔴 The day is the day where the USER is.
//
// This is the one part of occurrences that can be tested without a database, so
// it is tested without one — every DB-backed test in this repository begins with
// `if dsn == "" { t.Skip() }`, and a skipping control is what let a broken
// ON CONFLICT ship on 18.09.
//
// The failure it guards against is not theoretical: on 17.09 a due_date read
// through a UTC session looked a day early and I called it a defect in the
// product. That was only a misread. This one would be WRITTEN — a «done» tapped
// at 23:00 in Tokyo filed under tomorrow, a streak broken for reasons invisible
// to the person who kept it.
func TestLocalDayIsTheUsersDay(t *testing.T) {
	// 2026-09-18 21:30 UTC.
	at := time.Date(2026, time.September, 18, 21, 30, 0, 0, time.UTC)

	for _, c := range []struct {
		zone string
		want string
	}{
		{"UTC", "2026-09-18"},
		{"Europe/Moscow", "2026-09-19"}, // 00:30 the next day
		{"Asia/Tokyo", "2026-09-19"},    // 06:30 the next day
		{"America/New_York", "2026-09-18"},
	} {
		got := LocalDay(at, c.zone).Format("2006-01-02")
		if got != c.want {
			t.Errorf("LocalDay(21:30Z, %s) = %s, want %s", c.zone, got, c.want)
		}
	}
}

// An unknown zone falls back to UTC rather than to the server's local time.
//
// ⚠ Deliberate: the server's zone is an accident of deployment, and falling back
// to it would make the same account behave differently after a host move.
func TestLocalDayFallsBackToUTCNotTheServer(t *testing.T) {
	at := time.Date(2026, time.September, 18, 21, 30, 0, 0, time.UTC)
	if got := LocalDay(at, "Mars/Olympus").Format("2006-01-02"); got != "2026-09-18" {
		t.Errorf("unknown zone gave %s, want the UTC day 2026-09-18", got)
	}
}

// Midnight and the last second of a day both belong to that day — the boundary
// is where an off-by-one hides.
func TestLocalDayHoldsAtTheBoundaries(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	midnight := time.Date(2026, time.September, 19, 0, 0, 0, 0, loc)
	lastSecond := time.Date(2026, time.September, 19, 23, 59, 59, 0, loc)

	for _, at := range []time.Time{midnight, lastSecond} {
		if got := LocalDay(at, "Europe/Moscow").Format("2006-01-02"); got != "2026-09-19" {
			t.Errorf("LocalDay(%s) = %s, want 2026-09-19", at.Format(time.RFC3339), got)
		}
	}
}
