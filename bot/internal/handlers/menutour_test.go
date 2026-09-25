package handlers

import (
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// N1 of pass 3 (Denis 23.09): the last onboarding screen has «ℹ️ Что в меню».
// The tour takes the place of the screen it was asked from, like any
// explanation, and «« Назад» puts that screen back; each of its buttons opens
// the entrance it names.
func TestTheMenuTourReplacesTheScreenAndBackRestoresIt(t *testing.T) {
	h, fake, chat := quickHandler(t)
	pressOn(h, chat, keyboards.MenuTourOpen, screenUnderHelp())

	got := fake.last(t)
	if got.Method != "editMessageText" {
		t.Fatalf("the tour came as %s, want it in place of the screen", got.Method)
	}
	if !strings.Contains(got.Text, "Что в меню") || !strings.Contains(got.Markup, `"mo_tasks"`) {
		t.Errorf("not the tour: %q / %s", got.Text, got.Markup)
	}

	pressOn(h, chat, "help_x", screenUnderHelp())
	back := fake.last(t)
	if back.Method != "editMessageText" || back.Text != "Связать или перенести?" {
		t.Errorf("«Назад» gave %s %q, want the screen back", back.Method, back.Text)
	}
}

func TestAMenuTourButtonOpensItsEntrance(t *testing.T) {
	h, fake, chat := quickHandler(t)
	pressOn(h, chat, keyboards.MenuTourPrefix+keyboards.ScreenCreate, screenUnderHelp())

	got := fake.last(t)
	if !strings.Contains(got.Text, "Что создаём?") {
		t.Errorf("➕ in the tour answered %s %q, want the create screen", got.Method, got.Text)
	}
}
