package keyboards

import (
	"strings"
	"testing"
)

// The six entrances, in order. This list is the decision itself, so the test
// states it literally rather than deriving it from the code under test — a
// test whose expectation is computed from its subject cannot disagree with it.
var wantEntrances = []string{
	"🏠 Меню", "🗓 Календарь",
	"📅 События", "📋 Задачи",
	"➕ Создать", "⚙️ Настройки",
}

func TestMainMenuHasExactlySixEntrances(t *testing.T) {
	kb := MainMenu()
	var got []string
	for _, row := range kb.Keyboard {
		for _, b := range row {
			got = append(got, b.Text)
		}
	}
	if len(got) != len(wantEntrances) {
		t.Fatalf("reply keyboard has %d buttons, want %d: %v", len(got), len(wantEntrances), got)
	}
	for i := range wantEntrances {
		if got[i] != wantEntrances[i] {
			t.Errorf("button %d is %q, want %q", i, got[i], wantEntrances[i])
		}
	}
}

func TestMainMenuCarriesNoActions(t *testing.T) {
	// 🔴 The rule the six were chosen by: the reply keyboard holds ENTRANCES,
	// never actions. "New Task", "Note" and "New Event" used to sit here, which
	// is why every inline screen ended up pointing back at it in prose.
	banned := []string{"Note", "Заметк", "New Task", "New Event", "Stats", "Статист", "Planning", "Планир"}
	kb := MainMenu()
	for _, row := range kb.Keyboard {
		for _, b := range row {
			for _, bad := range banned {
				if strings.Contains(b.Text, bad) {
					t.Errorf("reply button %q is an action, not an entrance — it belongs inline", b.Text)
				}
			}
		}
	}
}

func TestHomeInlineCallbacksFitTelegramsBudget(t *testing.T) {
	for _, row := range HomeInline().InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
		}
	}
}
