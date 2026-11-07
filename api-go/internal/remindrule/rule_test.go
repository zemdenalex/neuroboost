package remindrule

import (
	"testing"
	"time"
)

var msk = time.FixedZone("MSK", 3*3600)

func at(day, hh, mm int) time.Time { return time.Date(2026, 11, day, hh, mm, 0, 0, msk) }

func fmtAll(ts []time.Time) []string {
	out := []string{}
	for _, t := range ts {
		out = append(out, t.In(msk).Format("02 15:04"))
	}
	return out
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Denis's own line, 07.11: «экзамен 30.11, напоминать каждый день в 12 и 16».
// Over the last three days of the window: both times every day, the deadline
// day included up to the deadline moment, nothing at or after it.
func TestDailyTimesRunUpToTheDeadlineMoment(t *testing.T) {
	r := Rule{EveryDays: 1, Times: []string{"16:00", "12:00"}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	got := fmtAll(r.Expand(at(30, 14, 0), msk, at(28, 0, 0), at(30, 23, 0)))
	want := []string{"28 12:00", "28 16:00", "29 12:00", "29 16:00", "30 12:00"}
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Every 2 days is counted back from the deadline day, so the deadline day is
// always a reminder day, whatever day the rule was set on.
func TestEveryKDaysIsAnchoredOnTheDeadlineDay(t *testing.T) {
	r := Rule{EveryDays: 2, Times: []string{"09:00"}}
	got := fmtAll(r.Expand(TaskDeadline(at(30, 0, 0), msk), msk, at(25, 0, 0), at(30, 23, 0)))
	want := []string{"26 09:00", "28 09:00", "30 09:00"}
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Every 5 minutes: only what falls in the scan window, aligned to the
// deadline so overlapping scans produce the same moments.
func TestIntervalOnlyInsideTheWindowAndAligned(t *testing.T) {
	r := Rule{EveryMinutes: 5}
	got := fmtAll(r.Expand(at(30, 12, 2), msk, at(10, 9, 0), at(10, 9, 11)))
	want := []string{"10 09:02", "10 09:07"}
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Past the deadline nothing fires.
	if n := len(r.Expand(at(10, 9, 0), msk, at(10, 9, 0), at(10, 10, 0))); n != 0 {
		t.Errorf("%d reminders at or after the deadline", n)
	}
}

func TestDateOnlyTaskDeadlineIsTheEndOfThatDay(t *testing.T) {
	if got := TaskDeadline(at(30, 0, 0), msk).Format("02 15:04"); got != "30 23:59" {
		t.Errorf("midnight due → %s, want 30 23:59", got)
	}
	if got := TaskDeadline(at(30, 14, 0), msk).Format("02 15:04"); got != "30 14:00" {
		t.Errorf("timed due moved to %s", got)
	}
}

func TestValidate(t *testing.T) {
	bad := []Rule{
		{},
		{EveryMinutes: 4},
		{EveryMinutes: 5, EveryDays: 1, Times: []string{"12:00"}},
		{EveryDays: 1},
		{EveryDays: 31, Times: []string{"12:00"}},
		{EveryDays: 1, Times: []string{"25:00"}},
	}
	for _, r := range bad {
		if err := r.Validate(); err == nil {
			t.Errorf("%+v accepted", r)
		}
	}
	ok := Rule{EveryDays: 1, Times: []string{"16:00", "9:00", "16:00"}}
	if err := ok.Validate(); err != nil || !same(ok.Times, []string{"09:00", "16:00"}) {
		t.Errorf("normalised to %v (%v)", ok.Times, err)
	}
}

// An all-day event stored as local midnight runs through its day.
func TestAllDayEventDeadlineIsTheEndOfItsDay(t *testing.T) {
	if got := AllDayDeadline(at(30, 0, 0), msk).Format("02 15:04"); got != "30 23:59" {
		t.Errorf("all-day 30th → %s, want 30 23:59", got)
	}
	utcMidnight := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC) // 03:00 in Moscow
	if got := AllDayDeadline(utcMidnight, msk).In(msk).Format("02 15:04"); got != "30 23:59" {
		t.Errorf("all-day 30th as UTC midnight → %s, want 30 23:59", got)
	}
}
