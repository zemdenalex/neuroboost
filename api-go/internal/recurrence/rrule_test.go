package recurrence

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// 🔴 The grammar is exactly what the store supports, and no more.
//
// Widening it here without widening the store produces a rule that parses and
// never fires — which is precisely how «каждый год» and «по вторникам» became
// silent one-off events before v0.4.11.2. The rejected list below is that
// defect, written down.
func TestParseAcceptsOnlyWhatTheStoreSupports(t *testing.T) {
	for _, ok := range []string{
		"FREQ=DAILY",
		"FREQ=WEEKLY;INTERVAL=2",
		"FREQ=MONTHLY;COUNT=10",
		"FREQ=DAILY;UNTIL=2026-12-01",
		"FREQ=MONTHLY;INTERVAL=12",
	} {
		if _, err := Parse(ok); err != nil {
			t.Errorf("Parse(%q) failed: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"FREQ=YEARLY",
		"FREQ=WEEKLY;BYDAY=TU",
		"FREQ=HOURLY",
		"",
		"nonsense",
		"FREQ=DAILY;INTERVAL=0",
		"FREQ=DAILY;UNTIL=01.12.2026",
	} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) was accepted; it would never fire and nobody would be told", bad)
		}
	}
}

func TestOccursWalksTheInterval(t *testing.T) {
	anchor := day(2026, time.September, 18)
	r, err := Parse("FREQ=DAILY;INTERVAL=3")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		d    time.Time
		want bool
	}{
		{anchor, true},
		{day(2026, time.September, 19), false},
		{day(2026, time.September, 21), true},
		{day(2026, time.September, 15), false}, // before the anchor there is no series
	} {
		if got := Occurs(r, anchor, c.d); got != c.want {
			t.Errorf("Occurs(%s) = %v, want %v", c.d.Format("02.01"), got, c.want)
		}
	}
}

func TestWeeklyKeepsTheWeekday(t *testing.T) {
	anchor := day(2026, time.September, 18) // Friday
	r, _ := Parse("FREQ=WEEKLY")
	if !Occurs(r, anchor, day(2026, time.September, 25)) {
		t.Error("the next Friday should occur")
	}
	if Occurs(r, anchor, day(2026, time.September, 24)) {
		t.Error("a Thursday occurred on a weekly Friday series")
	}
}

// ⚠ The 31st does not slide to the 30th. «Раз в месяц 31-го» is written knowing
// some months are skipped; inventing an occurrence would be answering a
// question the user did not ask.
func TestMonthlySkipsMonthsWithoutThatDay(t *testing.T) {
	anchor := day(2026, time.January, 31)
	r, _ := Parse("FREQ=MONTHLY")
	if Occurs(r, anchor, day(2026, time.February, 28)) {
		t.Error("the 31st slid to the 28th")
	}
	if !Occurs(r, anchor, day(2026, time.March, 31)) {
		t.Error("the 31st of March should occur")
	}
}

func TestCountAndUntilEndTheSeries(t *testing.T) {
	anchor := day(2026, time.September, 18)

	counted, _ := Parse("FREQ=DAILY;COUNT=3")
	if !Occurs(counted, anchor, day(2026, time.September, 20)) {
		t.Error("the third day is inside COUNT=3")
	}
	if Occurs(counted, anchor, day(2026, time.September, 21)) {
		t.Error("the fourth day is past COUNT=3")
	}

	until, _ := Parse("FREQ=DAILY;UNTIL=2026-09-20")
	if !Occurs(until, anchor, day(2026, time.September, 20)) {
		t.Error("UNTIL is inclusive of its own day")
	}
	if Occurs(until, anchor, day(2026, time.September, 21)) {
		t.Error("a day past UNTIL occurred")
	}
}

func TestNextFindsTheFollowingOccurrence(t *testing.T) {
	anchor := day(2026, time.September, 18)
	r, _ := Parse("FREQ=WEEKLY")
	got, ok := Next(r, anchor, day(2026, time.September, 18))
	if !ok || !got.Equal(day(2026, time.September, 25)) {
		t.Errorf("Next = %s (%v), want 25.09", got.Format("02.01"), ok)
	}

	// A finished series has no next, and says so instead of looping.
	done, _ := Parse("FREQ=DAILY;COUNT=1")
	if _, ok := Next(done, anchor, anchor); ok {
		t.Error("a COUNT=1 series reported a next occurrence")
	}
}

// 🔴 A nil rule is not an every-day rule. Reaching here with nil means a caller
// forgot to check a parse error, and answering «yes, it occurs» would turn that
// into reminders for a series that does not exist.
func TestNilRuleOccursNever(t *testing.T) {
	if Occurs(nil, day(2026, time.September, 18), day(2026, time.September, 18)) {
		t.Error("a nil rule reported an occurrence")
	}
	if _, ok := Next(nil, day(2026, time.September, 18), day(2026, time.September, 18)); ok {
		t.Error("a nil rule reported a next occurrence")
	}
}
