package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// 🔴 End to end: «отпуск с 14.10 по 29.10» is ONE all-day event whose end is the
// midnight after the 29th, in the user's zone — the convention a one-day
// all-day event already uses.
func TestSpanReachesTheAPIAsOneEvent(t *testing.T) {
	h, fake, chat, acc := onboardedHandler(t)
	h.startNewEventFlow(chat)
	from, fromMidnight := moscowDay(3)
	to, toMidnight := moscowDay(18)
	say(h, chat, "отпуск с "+from+" по "+to)

	if card := fake.last(t).Text; !strings.Contains(card, from+" – "+to+" · 16 дней") {
		t.Fatalf("the card does not show the span:\n%s", card)
	}
	press(h, chat, "dr_ok")

	var posts int
	for _, wr := range acc.writes {
		if wr.Method != "POST" || wr.Path != "/api/events" {
			continue
		}
		posts++
		// Local midnight in UTC; the end is the midnight after the last day.
		wantStart := fromMidnight.UTC().Format("2006-01-02T15:04:05Z")
		wantEnd := toMidnight.AddDate(0, 0, 1).UTC().Format("2006-01-02T15:04:05Z")
		if wr.Body["starts_at"] != wantStart || wr.Body["ends_at"] != wantEnd || wr.Body["all_day"] != true {
			t.Errorf("sent %v – %v all_day %v", wr.Body["starts_at"], wr.Body["ends_at"], wr.Body["all_day"])
		}
	}
	if posts != 1 {
		t.Errorf("%d events posted, want exactly one", posts)
	}
}

// Opened for editing, a span comes back as a span — not as its first day.
func TestSpanRoundTripsThroughEditing(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	allDay := draftFromEvent(api.Event{
		ID: "e1", Title: "отпуск", AllDay: true,
		StartsAt: "2026-10-13T21:00:00Z", EndsAt: "2026-10-29T21:00:00Z",
	}, loc)
	if allDay.D.EndDay.IsZero() || allDay.D.EndDay.Day() != 29 {
		t.Errorf("all-day span opened as ending %s", allDay.D.EndDay)
	}
	start, end := draftBounds(allDay)
	if start.UTC().Format(time.RFC3339) != "2026-10-13T21:00:00Z" || end.UTC().Format(time.RFC3339) != "2026-10-29T21:00:00Z" {
		t.Errorf("saving it back unchanged would send %s – %s", start.UTC(), end.UTC())
	}

	oneDay := draftFromEvent(api.Event{ID: "e2", Title: "x", AllDay: true,
		StartsAt: "2026-10-13T21:00:00Z", EndsAt: "2026-10-14T21:00:00Z"}, loc)
	if !oneDay.D.EndDay.IsZero() {
		t.Errorf("a one-day event opened as a span to %s", oneDay.D.EndDay)
	}
}
