package handlers

import (
	"strings"
	"testing"
)

// 🔴 Review 17.09, finding 1: text typed on the card wiped the draft with
// «Что-то пошло не так». Onboarding now TEACHES people to type rather than tap,
// so the dominant screen must survive it. Retyping the line is a correction.
func TestTypingOnTheCardRetypesTheLine(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "йога завтра 08:00")
	say(h, chat, "йога завтра 09:00")

	if h.store.GetOrCreate(chat).CurrentFlow != "new_event" {
		t.Fatalf("typing on the card ended the flow")
	}
	card := fake.last(t).Text
	if !strings.Contains(card, "09:00") || strings.Contains(card, "не так") {
		t.Errorf("the retyped line did not replace the draft:\n%s", card)
	}
}

// Finding 2: typing an end straight onto «Конец повтора».
func TestTypingOnTheRepeatEndScreenIsRead(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "йога каждую неделю завтра 08:00")
	press(h, chat, "dr_edit")
	press(h, chat, "dre_rend")
	say(h, chat, "10 раз")

	if card := fake.last(t).Text; !strings.Contains(card, "10 раз") {
		t.Errorf("«10 раз» typed on the end screen was not read:\n%s", card)
	}
}

// Typing on the reminder picker adds a time, the frequency picker takes a period.
func TestTypingOnPickersIsRead(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "йога завтра 08:00")
	press(h, chat, "dre_remind")
	say(h, chat, "45 минут")
	if !strings.Contains(fake.last(t).Markup, "dr_rem_45") {
		t.Errorf("a time typed on the reminder picker was not added: %s", fake.last(t).Markup)
	}

	press(h, chat, "dr_rem_done")
	press(h, chat, "dre_repeat")
	say(h, chat, "раз в 3 дня")
	if card := fake.last(t).Text; !strings.Contains(card, "раз в 3 дня") {
		t.Errorf("a period typed on the frequency picker was not read:\n%s", card)
	}
}

// Nothing typed anywhere in the flow may silently destroy the draft: an
// unreadable answer keeps it and says so.
func TestUnreadableTextKeepsTheDraft(t *testing.T) {
	h, _, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "йога каждую неделю завтра 08:00")
	press(h, chat, "dr_edit")
	say(h, chat, "ээээ")

	if _, ok := draftOf(h, chat); !ok {
		t.Errorf("text on the edit menu destroyed the draft")
	}
}

// The button keepDraft offers must lead back to something. In an open list with
// no single entry picked, «back to the card» would find no card and wipe the
// list — so it leads back to the list.
func TestTypingOnAListKeepsAWayBackToIt(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "среда\n10:00 завтрак\n12:00 обед")
	press(h, chat, "dr_many")
	say(h, chat, "ээээ")

	if markup := fake.last(t).Markup; !strings.Contains(markup, "dr_list") {
		t.Fatalf("the hint offers no way back to the list: %s", markup)
	}
	press(h, chat, "dr_list")
	if _, ok := listOf(h, chat); !ok {
		t.Errorf("going back destroyed the list")
	}
}
