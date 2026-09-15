package handlers

import (
	"os"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// 🔴 Denis, 15.09: «Если во время создания нажимается какая-то из кнопок меню,
// то создание должно прекратиться, а не думать что название пункта меню это
// название события».
//
// The defect was an ORDER, not a value: HandleMessage asked "is a flow
// running?" before it asked "is this a button?", so «🗓 Календарь» pressed
// mid-creation arrived as the event's title.
//
// That is why this is a source scan, the same way inplace_test.go is one. The
// failure does not live in a return value anywhere; it lives in which of two
// checks is written first, one level above every handler. A behavioural test
// would need a Telegram server to observe the difference.
func TestMenuButtonIsCheckedBeforeTheRunningFlow(t *testing.T) {
	src, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go: %v", err)
	}
	body, ok := funcBody(string(src), "func (h *Handler) HandleMessage(msg *tgbotapi.Message) {")
	if !ok {
		t.Fatal("HandleMessage not found — this scan is checking nothing")
	}

	menuAt := strings.Index(body, "keyboards.MenuScreen(")
	if menuAt < 0 {
		t.Fatal("HandleMessage never asks keyboards.MenuScreen whether the text is a " +
			"button press — a menu button pressed mid-flow becomes the event's title")
	}
	flowAt := strings.Index(body, `us.CurrentFlow != ""`)
	if flowAt < 0 {
		t.Fatal(`HandleMessage no longer checks us.CurrentFlow != "" — the scan has ` +
			"drifted from the code it guards")
	}
	if menuAt > flowAt {
		t.Errorf("the flow check comes before the menu check (%d > %d): a menu button "+
			"pressed during creation would be read as text again", menuAt, flowAt)
	}

	// Abandoning the flow is half of it. The other half is that the screen
	// actually opens — a guard that only clears the flow leaves the user
	// staring at nothing, and a test that only checks the clear passes on it.
	if !strings.Contains(body, "h.store.ClearFlow(chatID)") {
		t.Error("HandleMessage recognises a menu button but never clears the flow")
	}
	if !strings.Contains(body, "h.openScreen(chatID, screen)") {
		t.Error("HandleMessage recognises a menu button but never opens its screen")
	}
}

// Every screen a reply button can name has somewhere to go. Without this, a
// seventh button would clear the flow and then render nothing at all — which
// looks exactly like the bot swallowing the message.
func TestEveryMenuScreenHasACase(t *testing.T) {
	src, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go: %v", err)
	}
	body, ok := funcBody(string(src), "func (h *Handler) openScreen(chatID int64, screen string) {")
	if !ok {
		t.Fatal("openScreen not found — this scan is checking nothing")
	}

	screens := map[string]string{
		keyboards.ScreenMenu:     "keyboards.ScreenMenu",
		keyboards.ScreenCalendar: "keyboards.ScreenCalendar",
		keyboards.ScreenAgenda:   "keyboards.ScreenAgenda",
		keyboards.ScreenTasks:    "keyboards.ScreenTasks",
		keyboards.ScreenCreate:   "keyboards.ScreenCreate",
		keyboards.ScreenSettings: "keyboards.ScreenSettings",
	}
	// The map above is the literal expectation. Hold it against the keyboard
	// so the two cannot drift: a button whose screen is missing here would be
	// unguarded and this test would say so.
	for _, row := range keyboards.MainMenu().Keyboard {
		for _, b := range row {
			screen, known := keyboards.MenuScreen(b.Text)
			if !known {
				t.Errorf("reply button %q has no screen", b.Text)
				continue
			}
			if _, listed := screens[screen]; !listed {
				t.Errorf("screen %q (button %q) is not in this test's list", screen, b.Text)
			}
		}
	}

	for _, constName := range screens {
		if !strings.Contains(body, "case "+constName+":") {
			t.Errorf("openScreen has no case for %s", constName)
		}
	}
}
