package keyboards

import (
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// 🔴 Denis, 15.09: «Если во время создания нажимается какая-то из кнопок меню,
// то создание должно прекратиться, а не думать что название пункта меню это
// название события».
//
// The guard that does that needs to know which texts ARE menu buttons, and it
// lives in package handlers. Until now that knowledge existed only inside
// keyboards (as button text) and inside parity_test.go (as a literal map in a
// _test.go file, which handlers cannot import). So the set had to be exported —
// and derived from the keyboard itself, not retyped, or a seventh button would
// be added one day and guarded by nobody.
func TestMenuScreenCoversEveryReplyButton(t *testing.T) {
	seen := 0
	for _, row := range MainMenu(i18n.RU).Keyboard {
		for _, b := range row {
			seen++
			screen, ok := MenuScreen(b.Text)
			if !ok {
				t.Errorf("reply button %q has no screen — a press of it cannot "+
					"interrupt a running flow, so it would be read as text", b.Text)
				continue
			}
			if screen == "" {
				t.Errorf("reply button %q maps to an empty screen", b.Text)
			}
		}
	}
	if seen == 0 {
		t.Fatal("MainMenu carries no buttons — the test is checking nothing")
	}
}

func TestMenuScreenRejectsOrdinaryText(t *testing.T) {
	// The exact failure Denis hit: the label as the title of an event.
	for _, text := range []string{"Оркестр", "", "Календарь", "меню", "🗓"} {
		if screen, ok := MenuScreen(text); ok {
			t.Errorf("MenuScreen(%q) = %q, want no match", text, screen)
		}
	}
}

// Each label leads somewhere different: a table that collapses two buttons onto
// one screen would make the guard open the wrong one and look like a bug in the
// screen rather than in the table.
func TestMenuScreensAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, row := range MainMenu(i18n.RU).Keyboard {
		for _, b := range row {
			screen, _ := MenuScreen(b.Text)
			if prev, dup := seen[screen]; dup {
				t.Errorf("buttons %q and %q both lead to screen %q", prev, b.Text, screen)
			}
			seen[screen] = b.Text
		}
	}
}
