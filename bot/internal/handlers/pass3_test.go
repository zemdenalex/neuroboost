package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// Denis's third pass, 17.09 23:04 — ref/feedback/bot-prohod3-otvet-denisa-2026-09-17.md.

// 🔴 «Карточка одна но теперь непонятно что еще и задача была создана».
func TestCardSaysWhenATaskComesWithTheEvent(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "задача завтра в 15 позвонить в банк")

	card := fake.last(t).Text
	if !strings.Contains(card, "задача") {
		t.Errorf("the card does not say a task is created too:\n%s", card)
	}
}

// «стоматолог 1500» — a compact time written at the END of the line.
func TestCompactTimeAtTheEndOfALine(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "стоматолог 1500")
	press(h, chat, "qa_event")

	card := fake.last(t).Text
	if !strings.Contains(card, "15:00") {
		t.Errorf("«1500» at the end was not read as a time:\n%s", card)
	}
	if !strings.Contains(card, "стоматолог") || strings.Contains(card, "стоматолог 1500") {
		t.Errorf("the number stayed in the title:\n%s", card)
	}
}

// An all-day span must not look like a timed event on a button.
func TestPickerLabelsShowSpansAndAllDay(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	span := eventPickLabel(api.Event{
		Title: "отпуск", AllDay: true,
		StartsAt: "2026-10-13T21:00:00Z", EndsAt: "2026-10-29T21:00:00Z",
	}, loc)
	if !strings.Contains(span, "14.10") || !strings.Contains(span, "29.10") {
		t.Errorf("a 16-day holiday reads as %q — it looks like one day", span)
	}
	oneDay := eventPickLabel(api.Event{
		Title: "анализы", AllDay: true,
		StartsAt: "2026-10-13T21:00:00Z", EndsAt: "2026-10-14T21:00:00Z",
	}, loc)
	if strings.Contains(oneDay, "00:00") {
		t.Errorf("an all-day event reads as a midnight event: %q", oneDay)
	}
}

// Two dates with no «с … по» are a question, not a guess.
func TestTwoDatesAreAskedAbout(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "созвон 14.10 16.10")

	got := fake.last(t)
	for _, want := range []string{"dr_dspan", "dr_dpick"} {
		if !strings.Contains(got.Markup, want) {
			t.Errorf("two dates did not raise the question (%s): %s", want, got.Markup)
		}
	}

	press(h, chat, "dr_dspan")
	if card := fake.last(t).Text; !strings.Contains(card, "14.10 – 16.10") {
		t.Errorf("«с 14.10 по 16.10» was not applied:\n%s", card)
	}
}

// Three dates: a span, or the ones you tick.
func TestThreeDatesOfferASpanOrAChoice(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "тренировка 14.10 16.10 18.10")

	press(h, chat, "dr_dpick")
	pick := fake.last(t)
	if !strings.Contains(pick.Markup, "dr_dtog_0") || !strings.Contains(pick.Markup, "dr_dmake") {
		t.Fatalf("no per-date choice: %s", pick.Markup)
	}
	// Everything starts ticked; untick the middle date.
	press(h, chat, "dr_dtog_1")
	press(h, chat, "dr_dmake")

	// The list card writes days in words: «14 октября».
	list := fake.last(t).Text
	if !strings.Contains(list, "14 октября") || !strings.Contains(list, "18 октября") || strings.Contains(list, "16 октября") {
		t.Errorf("the ticked dates did not become the list:\n%s", list)
	}
}
