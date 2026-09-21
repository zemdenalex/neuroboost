package handlers

import (
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

const seriesEvent = `{"id":"e1","title":"планёрка","starts_at":"2026-10-20T09:00:00Z","ends_at":"2026-10-20T10:00:00Z","rrule":"FREQ=DAILY","tags":[]}`
const plainEvent = `{"id":"e1","title":"врач","starts_at":"2026-10-20T11:50:00Z","ends_at":"2026-10-20T13:00:00Z","color":"#f00","tags":[]}`

// The card is the API's dry run, not the bot's guess: the mapping lives in
// one place (spec §A1), and the losses it names are the ones shown.
func TestToTaskShowsTheDryRunBeforeWriting(t *testing.T) {
	h, fake, cf := convertAPI(t, `{"id":"t9","title":"врач","status":"TODO","priority":3}`, plainEvent)
	const chat = 910
	h.handleToTaskStart(chat, 0, "e1")
	h.handleToTaskStep(chat, 0, "e2m_m")
	card := fake.last(t)
	if !strings.Contains(cf.toTaskBody, `"dry_run":true`) {
		t.Fatalf("the card was not built from a dry run: %s", cf.toTaskBody)
	}
	if !strings.Contains(card.Text, "час начала") || !strings.Contains(card.Text, "цвет") {
		t.Errorf("losses not named on the card: %q", card.Text)
	}
	h.handleToTaskStep(chat, 0, "e2ok")
	if strings.Contains(cf.toTaskBody, `"dry_run":true`) {
		t.Errorf("✅ sent another dry run: %s", cf.toTaskBody)
	}
	if !strings.Contains(cf.toTaskBody, `"mode":"move"`) {
		t.Errorf("real call body = %s", cf.toTaskBody)
	}
}

// «Только этот раз» exists only when a DAY is known — an occurrence id.
func TestOnceIsOfferedOnlyForAnOccurrence(t *testing.T) {
	h, fake, _ := convertAPI(t, "", seriesEvent)
	h.handleToTaskStart(911, 0, "e1")
	h.handleToTaskStep(911, 0, "e2m_l")
	if strings.Contains(fake.last(t).Markup, "e2r_o") {
		t.Errorf("«this once» offered on the whole series: %s", fake.last(t).Markup)
	}
	// A crafted «once» on the series writes nothing.
	h.handleToTaskStep(911, 0, "e2r_o")

	h2, fake2, cf2 := convertAPI(t, "", seriesEvent)
	h2.handleToTaskStart(912, 0, "e1:2026-10-22")
	h2.handleToTaskStep(912, 0, "e2m_m")
	if !strings.Contains(fake2.last(t).Markup, "e2r_o") {
		t.Fatalf("no «this once» on an occurrence: %s", fake2.last(t).Markup)
	}
	h2.handleToTaskStep(912, 0, "e2r_o")
	if !strings.Contains(cf2.toTaskBody, `"repeat":"once"`) {
		t.Errorf("dry run body = %s", cf2.toTaskBody)
	}
}

func TestEveryLostCodeHasWords(t *testing.T) {
	for _, code := range []string{"start_time", "color", "location", "reminders"} {
		if l := lostLine(i18n.RU, code); l == "" || strings.Contains(l, code) || strings.Contains(l, "ещё кое-что") {
			t.Errorf("%s → %q: not worded", code, l)
		}
	}
}

// The button that starts it all sits on the event, carrying the id as it was
// opened — an occurrence stays an occurrence.
func TestTheEventOffersToBecomeATask(t *testing.T) {
	found := false
	eachButton(keyboards.EventEditor(i18n.RU, "e1", "e1:2026-10-22"), func(data string) {
		if data == "e2t_e1:2026-10-22" {
			found = true
		}
	})
	if !found {
		t.Error("EventEditor has no «Сделать задачей» for the occurrence it was opened on")
	}
	found = false
	eachButton(keyboards.EventCard(i18n.RU, "e1"), func(data string) {
		if data == "e2t_e1" {
			found = true
		}
	})
	if !found {
		t.Error("EventCard has no «Сделать задачей»")
	}
}
