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

// 🔴 A day is a date, not an instant.
//
// The anchor comes out of a DATE column as midnight UTC. «Сегодня» comes from
// LocalDay() as midnight in the user's zone. For Moscow that is three hours
// EARLIER than the same date's UTC midnight, so comparing instants said the
// anchor day itself was «before the series began», and the hour arithmetic put
// every later day one step short — a Monday task surfacing on Tuesdays.
//
// Found on 20.09 by the first test that created a repeating task through the
// API instead of inserting the row: until then both times in every test were
// built in one zone, and the seam between a DATE and a local midnight was never
// crossed.
func TestADayIsADateWhateverZoneItArrivesIn(t *testing.T) {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	anchorUTC := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC) // a Monday, as a DATE column returns it

	daily, _ := Parse("FREQ=DAILY")
	weekly, _ := Parse("FREQ=WEEKLY")
	everyOther, _ := Parse("FREQ=DAILY;INTERVAL=2")

	for _, loc := range []*time.Location{msk, ny, time.UTC} {
		day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, loc) }

		if !Occurs(daily, anchorUTC, day(2026, 9, 21)) {
			t.Errorf("%s: the anchor day itself is not in a daily series", loc)
		}
		if Occurs(daily, anchorUTC, day(2026, 9, 20)) {
			t.Errorf("%s: the day before the anchor is in the series", loc)
		}
		if !Occurs(weekly, anchorUTC, day(2026, 9, 28)) {
			t.Errorf("%s: the next Monday is not in a weekly series", loc)
		}
		if Occurs(weekly, anchorUTC, day(2026, 9, 29)) {
			t.Errorf("%s: a Monday series occurs on Tuesday", loc)
		}
		if Occurs(everyOther, anchorUTC, day(2026, 9, 22)) || !Occurs(everyOther, anchorUTC, day(2026, 9, 23)) {
			t.Errorf("%s: «через день» lands on the wrong days", loc)
		}
	}

	// ⚠ And across a daylight-saving change, where a local day is 23 or 25
	// hours long and dividing hours by 24 drops or invents a day.
	anchorNY := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	afterDST := time.Date(2026, 11, 6, 0, 0, 0, 0, ny) // clocks went back on 1.11
	if !Occurs(weekly, anchorNY, afterDST) {
		t.Error("a weekly series loses its day across the end of daylight saving")
	}
}
