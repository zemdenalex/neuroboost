package handlers

import (
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"

	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

func TestAgendaGroupsByDayAndSaysWhichDay(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	events := []api.Event{
		{Title: "Созвон", StartsAt: "2026-08-18T11:00:00Z"},
		{Title: "Ужин", StartsAt: "2026-08-19T16:00:00Z"},
	}
	got := agendaText(i18n.RU, events, now, "UTC")

	if !strings.Contains(got, "Сегодня") {
		t.Errorf("an event today is not labelled Сегодня:\n%s", got)
	}
	if !strings.Contains(got, "Завтра") {
		t.Errorf("an event tomorrow is not labelled Завтра:\n%s", got)
	}
	if strings.Index(got, "Созвон") > strings.Index(got, "Ужин") {
		t.Errorf("events are not in chronological order:\n%s", got)
	}
}

func TestAgendaSaysSoWhenThereIsNothing(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	got := agendaText(i18n.RU, nil, now, "UTC")
	if got == "" {
		t.Error("an empty agenda rendered an empty message — the screen would look broken")
	}
	if !strings.Contains(got, "Ничего") {
		t.Errorf("an empty agenda does not say it is empty:\n%s", got)
	}
}

func TestAgendaEscapesTitlesForHTML(t *testing.T) {
	// 🔴 The message is sent with ParseMode HTML. A title containing < or & is
	// not a hypothetical: it breaks the WHOLE message, so the user sees nothing
	// at all rather than one odd title.
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	events := []api.Event{{Title: "R&D <срочно>", StartsAt: "2026-08-18T11:00:00Z"}}
	got := agendaText(i18n.RU, events, now, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("the ampersand was not escaped:\n%s", got)
	}
}

// Denis, 23.09 (check of the pass-3 fixes, A4): «в списке событий нет конца».
// A line carries start–end, an all-day event says so, and the separator is not
// an em-dash (user-facing text rule).
func TestAgendaShowsWhenAnEventEnds(t *testing.T) {
	now := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	events := []api.Event{
		{Title: "работать", StartsAt: "2026-09-24T01:00:00Z", EndsAt: "2026-09-24T22:00:00Z"},
		{Title: "отпуск", StartsAt: "2026-09-24T00:00:00Z", EndsAt: "2026-09-25T00:00:00Z", AllDay: true},
	}
	got := agendaText(i18n.RU, events, now, "UTC")
	for _, want := range []string{"01:00–22:00", "весь день"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "—") {
		t.Errorf("an em-dash in user-facing text:\n%s", got)
	}
}

// A day heading further out names the weekday in the chat's language, not Go's
// English «Mon».
func TestAgendaDayHeadingsAreInRussian(t *testing.T) {
	now := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC) // Wednesday
	events := []api.Event{{Title: "концерт", StartsAt: "2026-09-26T18:00:00Z", EndsAt: "2026-09-26T20:00:00Z"}}
	got := agendaText(i18n.RU, events, now, "UTC")
	if strings.Contains(got, "Sat") || !strings.Contains(got, "сб") {
		t.Errorf("the heading is not in Russian:\n%s", got)
	}
}
