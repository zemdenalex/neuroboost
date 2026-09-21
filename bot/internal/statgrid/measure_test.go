package statgrid

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

func sp(s string) *string { return &s }

func TestAllDayEventsAreCountedNotSpanned(t *testing.T) {
	spans, allDay := EventSpans([]api.Event{
		{StartsAt: "2026-09-21T07:00:00Z", EndsAt: "2026-09-21T08:00:00Z"},
		{StartsAt: "2026-09-21T00:00:00Z", EndsAt: "2026-09-22T00:00:00Z", AllDay: true},
	})
	if len(spans) != 1 || allDay != 1 {
		t.Errorf("spans=%d allDay=%d", len(spans), allDay)
	}
}

// Spec §2 / Denis 22.09: estimate of what was closed; logged time wins; 30 min
// when there is neither.
func TestATaskIsWorthItsTime(t *testing.T) {
	if m := TaskMinutes(api.Task{EstimatedMinutes: 90}); m != 90 {
		t.Errorf("estimate: %d", m)
	}
	if m := TaskMinutes(api.Task{EstimatedMinutes: 90, ActualMinutes: 50}); m != 50 {
		t.Errorf("logged wins: %d", m)
	}
	if m := TaskMinutes(api.Task{}); m != 30 {
		t.Errorf("default: %d", m)
	}
}

// 🔴 A task closed at 00:30 Moscow belongs to that Moscow day, not to the
// previous UTC one — and a series day counts on its own date.
func TestTaskPointsLandOnTheUsersDay(t *testing.T) {
	pts := TaskPoints([]api.Task{
		{ID: "a", Status: "DONE", CompletedAt: "2026-09-21T21:30:00Z", EstimatedMinutes: 60},
		{ID: "s", Status: "TODO", Rrule: "FREQ=DAILY"},
		{ID: "open", Status: "TODO"},
	}, []api.TaskOccurrence{
		{TaskID: "s", Occurrence: "2026-09-22", State: "done"},
		{TaskID: "s", Occurrence: "2026-09-23", State: "skipped"},
	}, msk)
	day22 := Cell{From: time.Date(2026, 9, 22, 0, 0, 0, 0, msk), To: time.Date(2026, 9, 23, 0, 0, 0, 0, msk)}
	if got := SumPoints(pts, day22); got != 90*time.Minute {
		t.Errorf("22.09 = %v, want 1h30 (60 closed at 00:30 MSK + a 30-min series day)", got)
	}
	day23 := Cell{From: day22.To, To: day22.To.AddDate(0, 0, 1)}
	if got := SumPoints(pts, day23); got != 0 {
		t.Errorf("a skipped day counted: %v", got)
	}
}

func TestReflectionDaysAreLocal(t *testing.T) {
	days := ReflectionDays([]api.Reflection{{CreatedAt: "2026-09-21T22:10:00Z"}}, msk)
	if !days["2026-09-22"] || days["2026-09-21"] {
		t.Errorf("days = %v, want 22.09 in Moscow", days)
	}
}

// A series day has a date and no hour. In the week grid (columns are hours) it
// must not pile up in 00:00; it still counts toward the day.
func TestSeriesDaysAreMarkedAsHavingNoHour(t *testing.T) {
	pts := TaskPoints(nil, []api.TaskOccurrence{{TaskID: "s", Occurrence: "2026-09-22", State: "done"}}, msk)
	if len(pts) != 1 || !pts[0].DayOnly {
		t.Fatalf("points = %+v", pts)
	}
	c := Cell{From: time.Date(2026, 9, 22, 0, 0, 0, 0, msk), To: time.Date(2026, 9, 22, 1, 0, 0, 0, msk)}
	if SumPointsTimed(pts, c) != 0 || SumPoints(pts, c) == 0 {
		t.Error("a series day must count by day, never in the 00:00 hour")
	}
}
