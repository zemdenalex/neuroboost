package handlers

import (
	"strings"
	"testing"
)

// Проход Дениса по v0.4.11.3, 18.09 03:33.
//
// 🔴 «если начать создавать например задачу, и написать другую то он пишет что-то
// пошло не так». After showTaskCard the flow is new_task/card, and «card» was in
// no case of handleNewTaskFlow — so the second line fell to default, which
// clears the flow and apologises. Typing a second thing is the most ordinary
// act there is.
func TestASecondTaskLineReplacesTheFirst(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	if got := fake.last(t); !strings.Contains(got.Text, "помыться") {
		t.Fatalf("первая задача не показана: %q", got.Text)
	}

	say(h, chat, "задача на завтра выкинуть мусор")
	got := fake.last(t)
	if strings.Contains(got.Text, "Что-то пошло не так") {
		t.Fatalf("вторая строка ответила ошибкой: %q", got.Text)
	}
	if !strings.Contains(got.Text, "мусор") {
		t.Errorf("вторая строка не заменила карточку: %q", got.Text)
	}
}

// 🔴 The same line twice — exactly what Denis typed — must also work. This is
// the literal reproduction from the log.
func TestTheSameTaskLineTwiceDoesNotApologise(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	say(h, chat, "задача на завтра помыться")

	if got := fake.last(t); strings.Contains(got.Text, "Что-то пошло не так") {
		t.Errorf("повтор той же строки ответил ошибкой: %q", got.Text)
	}
}

// 🔴 «И это работает даже если на этапе создания нажать отмена, он вернется в
// меню, но следующая задача все равно не создастся». Cancel drew the menu and
// left the flow running, so the NEXT line was still read as an answer to a
// question that was no longer on screen.
func TestCancelReallyClearsTheFlow(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	press(h, chat, "main_menu")

	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "" {
		t.Errorf("отмена оставила флоу %q — следующая строка уедет в него", flow)
	}

	say(h, chat, "задача на завтра выкинуть мусор")
	if got := fake.last(t); strings.Contains(got.Text, "Что-то пошло не так") {
		t.Errorf("после отмены следующая задача ответила ошибкой: %q", got.Text)
	}
}
