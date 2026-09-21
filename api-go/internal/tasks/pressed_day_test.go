package tasks

import (
	"testing"
	"time"

	"neuroboost/api-go/internal/recurrence"
)

// What does «✅ готово» mean on a series that has not started yet?
//
// 🔴 The case that broke Denis's 21.09 pass, and the reason this is a pure
// function rather than a branch inside the handler: the defect needed a live
// database, a JWT and a Telegram button to reproduce, and it is one line of
// arithmetic. The handler test would have proved the wiring; this one proves
// the meaning.
func mustRule(t *testing.T, s string) *recurrence.Rule {
	t.Helper()
	r, err := recurrence.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return r
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestAPressOnADayInTheSeriesMeansThatDay(t *testing.T) {
	rule := mustRule(t, "FREQ=DAILY")
	anchor := day(2026, time.September, 21)

	got, ok := PressedDay(rule, anchor, day(2026, time.September, 21))
	if !ok {
		t.Fatal("a daily series anchored today has today in it")
	}
	if !got.Equal(day(2026, time.September, 21)) {
		t.Errorf("got %s, want 2026-09-21", got.Format("2006-01-02"))
	}
}

// Denis, 21.09: «позвонить в банк завтра 30м !1 каждый день» → the card was
// right, the button was not.
func TestAPressBeforeTheSeriesStartsMeansItsFirstDay(t *testing.T) {
	rule := mustRule(t, "FREQ=DAILY")
	anchor := day(2026, time.September, 22) // due date: tomorrow
	today := day(2026, time.September, 21)

	if recurrence.Occurs(rule, anchor, today) {
		t.Fatal("precondition: today must NOT be in this series, or the test proves nothing")
	}

	got, ok := PressedDay(rule, anchor, today)
	if !ok {
		t.Fatal("the series starts tomorrow; a press must resolve to it, not be refused")
	}
	if !got.Equal(day(2026, time.September, 22)) {
		t.Errorf("got %s, want 2026-09-22", got.Format("2006-01-02"))
	}
}

// A weekly Monday series, pressed on a Wednesday, closes the coming Monday —
// not the one that already passed. Closing the past would silently rewrite
// history the user did not ask about.
func TestAPressBetweenOccurrencesLooksForward(t *testing.T) {
	rule := mustRule(t, "FREQ=WEEKLY")
	anchor := day(2026, time.September, 21) // a Monday
	got, ok := PressedDay(rule, anchor, day(2026, time.September, 23))
	if !ok {
		t.Fatal("a weekly series always has a next day")
	}
	if !got.Equal(day(2026, time.September, 28)) {
		t.Errorf("got %s, want 2026-09-28", got.Format("2006-01-02"))
	}
}

// 🔴 The refusal must survive. A series that has run out has no day to close,
// and answering with some day anyway would be the worse bug — it would file
// work against a series that is over.
func TestAPressOnAnExhaustedSeriesStillHasNoDay(t *testing.T) {
	rule := mustRule(t, "FREQ=DAILY;COUNT=2")
	anchor := day(2026, time.September, 1)

	if _, ok := PressedDay(rule, anchor, day(2026, time.September, 10)); ok {
		t.Error("a two-day series that ended on 02.09 must not resolve a press made on 10.09")
	}
}

func TestAPressWithoutARuleHasNoDay(t *testing.T) {
	if _, ok := PressedDay(nil, day(2026, time.September, 1), day(2026, time.September, 1)); ok {
		t.Error("no rule means no series means no day")
	}
}
