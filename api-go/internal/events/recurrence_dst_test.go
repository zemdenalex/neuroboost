package events

import (
	_ "time/tzdata" // embed the IANA tz database so LoadLocation works on any host/CI image
	"testing"
	"time"
)

func mkEventTZ(id string, start time.Time, dur time.Duration, rrule, tz string) Event {
	e := mkEvent(id, start, dur, rrule)
	e.Timezone = tz
	return e
}

func localHour(t time.Time, loc *time.Location) int { return t.In(loc).Hour() }

// A recurring "09:00 local" event must stay at 09:00 local across the spring-forward
// DST transition — not drift to 10:00 because the underlying UTC instant was advanced
// by a fixed 24h. EU spring-forward 2026: Sun 2026-03-29 02:00→03:00 (CET→CEST).
func TestExpandDailyPreservesLocalTimeAcrossSpringForward(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin: %v", err)
	}
	start := time.Date(2026, 3, 27, 9, 0, 0, 0, berlin).UTC() // 09:00 Berlin, stored as UTC
	e := mkEventTZ("dst-spring", start, time.Hour, "FREQ=DAILY;COUNT=5", "Europe/Berlin")
	got := expandRecurrence(e, dt(2026, 3, 1, 0, 0), dt(2026, 4, 15, 0, 0), nil)
	if len(got) != 5 {
		t.Fatalf("want 5 instances, got %d (%v)", len(got), dates(got))
	}
	for i, inst := range got {
		if h := localHour(inst.StartsAt, berlin); h != 9 {
			t.Errorf("instance %d (%s) local hour = %d, want 9 — DST drift", i, dates(got)[i], h)
		}
	}
}

// Same invariant across the fall-back transition (drifts the opposite direction).
// EU fall-back 2026: Sun 2026-10-25 03:00→02:00 (CEST→CET).
func TestExpandDailyPreservesLocalTimeAcrossFallBack(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin: %v", err)
	}
	start := time.Date(2026, 10, 23, 9, 0, 0, 0, berlin).UTC()
	e := mkEventTZ("dst-fall", start, time.Hour, "FREQ=DAILY;COUNT=5", "Europe/Berlin")
	got := expandRecurrence(e, dt(2026, 10, 1, 0, 0), dt(2026, 11, 15, 0, 0), nil)
	if len(got) != 5 {
		t.Fatalf("want 5 instances, got %d (%v)", len(got), dates(got))
	}
	for i, inst := range got {
		if h := localHour(inst.StartsAt, berlin); h != 9 {
			t.Errorf("instance %d (%s) local hour = %d, want 9 — DST drift", i, dates(got)[i], h)
		}
	}
}

// Exception-skip must still match the right occurrence when the instant moves across
// DST. Guards the deliberate choice to keep ID / exception comparison on the UTC-date
// basis while the underlying instant is timezone-corrected.
func TestExpandExceptionSkipWorksAcrossDST(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin: %v", err)
	}
	start := time.Date(2026, 3, 28, 9, 0, 0, 0, berlin).UTC()
	e := mkEventTZ("dst-exc", start, time.Hour, "FREQ=DAILY;COUNT=3", "Europe/Berlin")
	exc := []time.Time{dt(2026, 3, 29, 0, 0)} // skip the DST-transition day
	got := expandRecurrence(e, dt(2026, 3, 1, 0, 0), dt(2026, 4, 15, 0, 0), exc)
	want := []string{"2026-03-28", "2026-03-30"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("exception across DST; dates = %v, want %v", g, want)
	}
	for i, inst := range got {
		if h := localHour(inst.StartsAt, berlin); h != 9 {
			t.Errorf("instance %d local hour = %d, want 9", i, h)
		}
	}
}

// A non-DST timezone (Moscow, fixed UTC+3) behaves correctly — the timezone-aware
// path must not misbehave for zones without transitions.
func TestExpandNonDSTTimezoneUnaffected(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("load Europe/Moscow: %v", err)
	}
	start := time.Date(2026, 3, 27, 9, 0, 0, 0, moscow).UTC()
	e := mkEventTZ("msk", start, time.Hour, "FREQ=DAILY;COUNT=5", "Europe/Moscow")
	got := expandRecurrence(e, dt(2026, 3, 1, 0, 0), dt(2026, 4, 15, 0, 0), nil)
	if len(got) != 5 {
		t.Fatalf("want 5 instances, got %d", len(got))
	}
	for i, inst := range got {
		if h := localHour(inst.StartsAt, moscow); h != 9 {
			t.Errorf("instance %d local hour = %d, want 9", i, h)
		}
	}
}
