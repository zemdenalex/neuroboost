package keyboards

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTaskCardSaveIsFirstAndAlwaysEnabled(t *testing.T) {
	kb := TaskCard()
	if len(kb.InlineKeyboard) == 0 || len(kb.InlineKeyboard[0]) == 0 {
		t.Fatal("TaskCard has no rows")
	}
	first := kb.InlineKeyboard[0][0]
	if first.CallbackData == nil || *first.CallbackData != "nt_save" {
		t.Errorf("expected the first button to be nt_save, got %+v", first)
	}
}

func TestTaskCardHasCancel(t *testing.T) {
	kb := TaskCard()
	found := false
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData != nil && *btn.CallbackData == "main_menu" {
				found = true
			}
		}
	}
	if !found {
		t.Error("TaskCard has no way back to the main menu")
	}
}

// Denis, 18.08: nothing is required. Both escapes must exist at every step —
// "⏭ Пропустить" for this field, "✅ Создать сейчас" for all remaining ones.
func TestEveryWizardKeyboardOffersBothEscapes(t *testing.T) {
	kbs := map[string]tgbotapi.InlineKeyboardMarkup{
		"priority": WizardPriority(),
		"due":      WizardDue(),
		"estimate": WizardEstimate(),
	}
	for name, kb := range kbs {
		var skip, save bool
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && *b.CallbackData == "nt_skip" {
					skip = true
				}
				if b.CallbackData != nil && *b.CallbackData == "nt_save" {
					save = true
				}
				if b.CallbackData != nil && len(*b.CallbackData) > 64 {
					t.Errorf("%s: callback_data %q is %d bytes", name, *b.CallbackData, len(*b.CallbackData))
				}
			}
		}
		if !skip {
			t.Errorf("%s step has no ⏭ Пропустить — the field became required", name)
		}
		if !save {
			t.Errorf("%s step has no ✅ Создать сейчас — the wizard became a gate", name)
		}
	}
}
