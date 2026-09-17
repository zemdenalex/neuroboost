package handlers

import (
	"strings"
	"testing"
)

// 🔴 Denis, 17.09: «тут просто даты перепутаны местами, он должен это понять и
// предложить в подтверждении их поменять местами или написать свои» — and the
// day question offered only today/tomorrow/the day after, with no way to type.
func TestBackwardsSpanOffersASwap(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "отпуск с 29.10 по 14.10")
	press(h, chat, "dr_ok")

	ask := fake.last(t)
	if !strings.Contains(ask.Markup, "dr_swap") {
		t.Fatalf("no swap offered: %s / %q", ask.Markup, ask.Text)
	}
	if !strings.Contains(ask.Text, "14.10") || !strings.Contains(ask.Text, "29.10") {
		t.Errorf("the question does not show both dates: %q", ask.Text)
	}

	press(h, chat, "dr_swap")
	card := fake.last(t).Text
	if !strings.Contains(card, "14.10 – 29.10") || strings.Contains(card, "⚠ проверь") {
		t.Errorf("after the swap the card is still wrong:\n%s", card)
	}
}

// Every day question takes a typed date, and says so with a button.
func TestDayQuestionOffersYourOwnDate(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "стоматолог 15:00")
	press(h, chat, "dr_ok")

	if m := fake.last(t).Markup; !strings.Contains(m, "dr_daytext") {
		t.Fatalf("the day question has no «своя дата»: %s", m)
	}
	press(h, chat, "dr_daytext")
	say(h, chat, "с 14.10 по 16.10")
	if card := fake.last(t).Text; !strings.Contains(card, "14.10 – 16.10") {
		t.Errorf("a span typed at the day question was not read:\n%s", card)
	}
}
