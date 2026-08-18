package handlers

import (
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
	got := agendaText(events, now, "UTC")

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
	got := agendaText(nil, now, "UTC")
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
	got := agendaText(events, now, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("the ampersand was not escaped:\n%s", got)
	}
}
