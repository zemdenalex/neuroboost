package handlers

import (
	"strings"
	"testing"
)

// Denis, 17.09: «когда он предлагает создать список, он все время пишет больше,
// чем реально потом создает» — «список из 6» for three events, because day
// headers were counted as entries. The number offered is the number made.
func TestListQuestionCountsEntriesNotLines(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "monday\n10 wake up\ntuesday\n12:40-13:30 call\nfriday\n13:30 -15 exam")

	if q := fake.last(t).Text; !strings.Contains(q, "из 3") {
		t.Errorf("the question does not say 3: %q", q)
	}
}

// 🔴 Denis, 17.09 (F5): a span ending before it starts carried ⚠, but ✅ created
// it anyway. Confirming asks for the dates again.
func TestBackwardsSpanIsAskedAgainOnConfirm(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "отпуск с 29.10 по 14.10")
	press(h, chat, "dr_ok")

	if fake.called_("createEvent") || !strings.Contains(fake.last(t).Text, "день") {
		t.Errorf("a backwards span was not asked again: %q", fake.last(t).Text)
	}
	if h.store.GetOrCreate(chat).FlowStep != "edit:date" {
		t.Errorf("step after ✅ = %q, want edit:date", h.store.GetOrCreate(chat).FlowStep)
	}
}
