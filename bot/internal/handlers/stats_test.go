package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

func msk() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}

func ev(startsAt, endsAt string, allDay bool) api.Event {
	return api.Event{StartsAt: startsAt, EndsAt: endsAt, AllDay: allDay}
}

// Friday 18.09.2026 12:00 Moscow. The ISO week runs Mon 14.09 – Sun 20.09.
func friday() time.Time {
	return time.Date(2026, 9, 18, 12, 0, 0, 0, msk())
}

// 🔴 Denis, 18.09: «статистика не работает». It never did — the screen was a
// stub that printed «Скоро». These are the numbers that replaced it, and they
// are worth exactly as much as their arithmetic.
func TestStatsCountsTheCurrentWeek(t *testing.T) {
	events := []api.Event{
		ev("2026-09-14T06:00:00Z", "2026-09-14T07:30:00Z", false), // Mon, 1.5h
		ev("2026-09-18T06:00:00Z", "2026-09-18T08:00:00Z", false), // Fri, 2h
		ev("2026-09-18T09:00:00Z", "2026-09-18T09:00:00Z", true),  // Fri, all-day
		// 🔴 Outside the week on both sides — these are what a naive "take
		// everything the API returned" would wrongly include.
		ev("2026-09-13T06:00:00Z", "2026-09-13T07:00:00Z", false), // Sun before
		ev("2026-09-21T06:00:00Z", "2026-09-21T07:00:00Z", false), // Mon after
	}

	s := BuildStats(events, nil, friday(), msk())

	if s.Events != 3 {
		t.Errorf("Events = %d, want 3 (two outside the week must not count)", s.Events)
	}
	if s.Hours < 3.4 || s.Hours > 3.6 {
		t.Errorf("Hours = %.2f, want 3.5 — all-day events must add no hours", s.Hours)
	}
	if s.AllDay != 1 {
		t.Errorf("AllDay = %d, want 1", s.AllDay)
	}
	if s.PerDay[0] != 1 || s.PerDay[4] != 2 {
		t.Errorf("PerDay = %v, want Monday 1 and Friday 2", s.PerDay)
	}
	if s.Busiest != 4 {
		t.Errorf("Busiest = %d, want 4 (Friday)", s.Busiest)
	}
}

// 🔴 A task due TODAY is not late. Measuring against `now` instead of the start
// of the day would paint the whole morning red.
func TestStatsDoesNotCallTodaysTasksOverdue(t *testing.T) {
	tasks := []api.Task{
		{Status: "TODO", DueDate: "2026-09-18T20:00:00Z"}, // today, later
		{Status: "TODO", DueDate: "2026-09-18T00:00:00Z"}, // today, midnight
		{Status: "TODO", DueDate: "2026-09-16T00:00:00Z"}, // two days ago
		{Status: "TODO"}, // no date, never late
	}

	s := BuildStats(nil, tasks, friday(), msk())

	if s.TasksOpen != 4 {
		t.Errorf("TasksOpen = %d, want 4", s.TasksOpen)
	}
	if s.TasksOverdue != 1 {
		t.Errorf("TasksOverdue = %d, want 1 — only the one from two days ago", s.TasksOverdue)
	}
}

// «Закрыто за неделю» answers "did this week go anywhere". An all-time total
// would be a number that only ever grows and says nothing.
func TestStatsCountsOnlyThisWeeksClosures(t *testing.T) {
	tasks := []api.Task{
		{Status: "DONE", CompletedAt: "2026-09-16T10:00:00Z"}, // this week
		{Status: "DONE", CompletedAt: "2026-09-01T10:00:00Z"}, // long ago
		{Status: "DONE"},      // closed, date unknown
		{Status: "CANCELLED"}, // neither open nor done
	}

	s := BuildStats(nil, tasks, friday(), msk())

	if s.TasksDone != 1 {
		t.Errorf("TasksDone = %d, want 1", s.TasksDone)
	}
	if s.TasksOpen != 0 {
		t.Errorf("TasksOpen = %d, want 0 — cancelled is not open", s.TasksOpen)
	}
}

// An empty week says so instead of printing a wall of zeroes.
func TestStatsSaysWhenThereIsNothing(t *testing.T) {
	s := BuildStats(nil, nil, friday(), msk())
	text := renderStats(i18n.RU, s, friday())
	if !strings.Contains(text, "Пока пусто") {
		t.Errorf("empty week does not say so:\n%s", text)
	}
	if strings.Contains(text, "Событий:") {
		t.Errorf("empty week printed a table of zeroes:\n%s", text)
	}
}

// And a week with content prints real numbers in both languages.
func TestStatsRendersNumbersInBothLanguages(t *testing.T) {
	events := []api.Event{ev("2026-09-18T06:00:00Z", "2026-09-18T08:00:00Z", false)}
	s := BuildStats(events, nil, friday(), msk())

	ru := renderStats(i18n.RU, s, friday())
	if !strings.Contains(ru, "Событий:") && strings.Contains(ru, "1") {
		t.Errorf("russian stats missing numbers:\n%s", ru)
	}
	en := renderStats(i18n.EN, s, friday())
	if !strings.Contains(en, "Events:") && strings.Contains(en, "1") {
		t.Errorf("english stats missing numbers:\n%s", en)
	}
	if ru == en {
		t.Error("both languages render identically; lang is ignored")
	}
}
