package keyboards

import "testing"

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
