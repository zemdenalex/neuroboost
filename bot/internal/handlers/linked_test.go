package handlers

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Denis 21.09: a scheduled task stays in the list, marked with its time.
func TestAScheduledTaskIsStillOpen(t *testing.T) {
	got := openTasks([]api.Task{
		{ID: "a", Status: "TODO"}, {ID: "b", Status: "SCHEDULED"},
		{ID: "c", Status: "DONE"}, {ID: "d", Status: "CANCELLED"},
	})
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("open = %+v, want a and b", got)
	}
}

func sp(s string) *string { return &s }

func TestTheNearestUpcomingLinkedEventWins(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	got := linkedEvents([]api.Event{
		{ID: "old", TaskID: sp("t1"), StartsAt: "2026-10-19T09:00:00Z"}, // yesterday: ignored
		{ID: "far", TaskID: sp("t1"), StartsAt: "2026-10-27T09:00:00Z"},
		{ID: "near", TaskID: sp("t1"), StartsAt: "2026-10-22T15:00:00Z"},
		{ID: "none", StartsAt: "2026-10-21T09:00:00Z"},
	}, now)
	if got["t1"].ID != "near" || len(got) != 1 {
		t.Errorf("linked = %+v, want t1 → near only", got)
	}
}

// Today's event still counts after its hour has passed: «запланирована на
// сегодня 09:00» is true at noon, and dropping it would hide the link.
func TestTodaysLinkedEventCountsAllDay(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	got := linkedEvents([]api.Event{{ID: "e", TaskID: sp("t1"), StartsAt: "2026-10-20T09:00:00Z"}}, now)
	if got["t1"].ID != "e" {
		t.Errorf("today's event was dropped: %+v", got)
	}
}

func TestWhenShortNamesTheDayTheWayPeopleDo(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) // Tuesday
	for _, c := range []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC), "сегодня 15:00"},
		{time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC), "завтра 09:30"},
		{time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC), "пт 15:00"},
		{time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC), "02.11 15:00"},
	} {
		if got := whenShort(i18n.RU, c.at, now); got != c.want {
			t.Errorf("%v: %q, want %q", c.at, got, c.want)
		}
	}
}
