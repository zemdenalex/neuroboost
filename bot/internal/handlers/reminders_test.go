package handlers

import (
	"strings"
	"testing"
)

// cardWithDraft opens a confirmation card for one event.
func cardWithDraft(t *testing.T) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "стоматолог завтра 15:00")
	if _, ok := draftOf(h, chat); !ok {
		t.Fatalf("no draft — the rest proves nothing")
	}
	return h, fake, chat
}

func offsetsOf(t *testing.T, h *Handler, chat int64) []int {
	t.Helper()
	st, _ := draftOf(h, chat)
	if st.ReminderOffsets == nil {
		return nil
	}
	return *st.ReminderOffsets
}

// 🔴 Denis, 16.09: «Упоминания должны быть множественным выбором (кроме не
// напоминать — снимает все), то есть на каждом можно галочку, а также можно
// свое время уведомлений».
func TestRemindersAreTicksNotOneChoice(t *testing.T) {
	h, fake, chat := cardWithDraft(t)
	press(h, chat, "dre_remind")
	press(h, chat, "dr_rem_10")
	press(h, chat, "dr_rem_60")

	got := offsetsOf(t, h, chat)
	if len(got) != 2 || !containsInt(got, 10) || !containsInt(got, 60) {
		t.Fatalf("two ticks left %v, want [10 60]", got)
	}
	// A tick keeps the picker open with both marked.
	markup := fake.last(t).Markup
	if strings.Count(markup, "✅") != 2 {
		t.Errorf("the picker does not show both ticks: %s", markup)
	}

	press(h, chat, "dr_rem_10")
	if got := offsetsOf(t, h, chat); len(got) != 1 || got[0] != 60 {
		t.Errorf("unticking left %v, want [60]", got)
	}
}

func TestNoReminderClearsEveryTick(t *testing.T) {
	h, _, chat := cardWithDraft(t)
	press(h, chat, "dre_remind")
	press(h, chat, "dr_rem_10")
	press(h, chat, "dr_rem_1440")
	press(h, chat, "dr_rem_none")

	st, _ := draftOf(h, chat)
	if st.ReminderOffsets == nil || len(*st.ReminderOffsets) != 0 {
		t.Errorf("«Не напоминать» left %v, want an explicit empty list", st.ReminderOffsets)
	}
}

// Its own time, typed — and it joins the ticks rather than replacing them.
func TestCustomReminderTimeIsTyped(t *testing.T) {
	h, fake, chat := cardWithDraft(t)
	press(h, chat, "dre_remind")
	press(h, chat, "dr_rem_10")
	press(h, chat, "dr_rem_custom")
	say(h, chat, "за 2 часа")

	got := offsetsOf(t, h, chat)
	if len(got) != 2 || !containsInt(got, 120) {
		t.Fatalf("typed «за 2 часа» left %v, want [10 120]", got)
	}
	if !strings.Contains(fake.last(t).Markup, "dr_rem_120") {
		t.Errorf("the typed time is not a tickable button: %s", fake.last(t).Markup)
	}

	press(h, chat, "dr_rem_done")
	card := fake.last(t).Text
	if !strings.Contains(card, "2 ч") {
		t.Errorf("the card does not show the typed reminder:\n%s", card)
	}
}

func containsInt(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
