package handlers

import (
	"strings"
	"testing"
)

// 🔴 Denis, 16.09: «он не подтверждает а сразу создает, надо чтобы подтверждал
// как с событиями». A task list now stops at a card, and «Список из 3» creates
// nothing by itself.
func TestTaskListShowsACardBeforeCreating(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewTaskFlow(chat)
	say(h, chat, "1. Отжаться 30м\n2. Подтянуться !1\n3. Присесть")
	press(h, chat, "dr_many")

	if fake.called_("createTask") {
		t.Fatal("the list was created without confirmation")
	}
	card := fake.last(t)
	for _, want := range []string{"Отжаться", "Подтянуться", "Присесть", "30m", "dr_makeall", "dr_pick"} {
		if !strings.Contains(card.Text+card.Markup, want) {
			t.Errorf("the list card is missing %q:\n%s\n%s", want, card.Text, card.Markup)
		}
	}
	// The estimate was read, not left in the title.
	if strings.Contains(card.Text, "Отжаться 30м") {
		t.Errorf("«30м» stayed in the title: %s", card.Text)
	}
}

// Removing one entry leaves the others, renumbered.
func TestTaskListDropsAnEntry(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewTaskFlow(chat)
	say(h, chat, "Отжаться\nПодтянуться\nПрисесть")
	press(h, chat, "dr_many")
	press(h, chat, "dr_item_1")
	press(h, chat, "dr_tdel_1")

	card := fake.last(t).Text
	if strings.Contains(card, "Подтянуться") {
		t.Errorf("the removed task is still on the card:\n%s", card)
	}
	if !strings.Contains(card, "2. ") || !strings.Contains(card, "Присесть") {
		t.Errorf("the rest of the list is not renumbered:\n%s", card)
	}
}

// Rewriting a line reads it with the task vocabulary again.
func TestTaskListRewritesAnEntry(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewTaskFlow(chat)
	say(h, chat, "Отжаться\nПодтянуться")
	press(h, chat, "dr_many")
	press(h, chat, "dr_item_0")
	press(h, chat, "dr_trew_0")
	say(h, chat, "Отжаться 50 раз 1ч !2")

	card := fake.last(t).Text
	if !strings.Contains(card, "Отжаться 50 раз") || !strings.Contains(card, "1h") {
		t.Errorf("the rewritten entry was not re-read:\n%s", card)
	}
	if !strings.Contains(card, "Подтянуться") {
		t.Errorf("rewriting one entry lost the other:\n%s", card)
	}
}
