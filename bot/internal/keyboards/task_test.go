package keyboards

import (
	"strings"
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
		"priority": WizardPriority(nil),
		"due":      WizardDue(""),
		"estimate": WizardEstimate(nil),
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

// Priority is INVERTED (1 Emergency .. 5 If Possible, 0 Buffer). A bare digit
// gives the reader no way to tell which end is urgent — the word is the only
// cue. Denis' priority keyboard (TaskPriority, now removed) always carried it;
// the wizard's must too.
func TestWizardPriorityLabelsCarryWords(t *testing.T) {
	kb := WizardPriority(nil)
	for _, want := range []string{"Emergency", "Urgent", "Normal", "Low", "If Possible", "Buffer"} {
		found := false
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if strings.Contains(b.Text, want) {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("no wizard priority button names %q — a bare digit does not say which end is urgent", want)
		}
	}
}

func TestWizardPriorityLayoutIsTwoRowsOfThree(t *testing.T) {
	kb := WizardPriority(nil)
	if len(kb.InlineKeyboard) < 2 || len(kb.InlineKeyboard[0]) != 3 || len(kb.InlineKeyboard[1]) != 3 {
		t.Fatalf("wizard priority layout changed shape: %d rows, first two have %d/%d buttons",
			len(kb.InlineKeyboard), len(kb.InlineKeyboard[0]), len(kb.InlineKeyboard[1]))
	}
}

// Spec, part 3: a step whose value is already known shows it as the current
// value, not silently skipped. The keyboard is where that has to be visible —
// mark the button matching the known value.
func TestWizardKeyboardsMarkTheKnownValue(t *testing.T) {
	p := 2
	kb := WizardPriority(&p)
	marked := 0
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if strings.HasPrefix(b.Text, "✓") {
				marked++
				if b.CallbackData == nil || *b.CallbackData != "nt_p_2" {
					t.Errorf("the marked button is %v, want nt_p_2", b)
				}
			}
		}
	}
	if marked != 1 {
		t.Errorf("WizardPriority(2) marked %d buttons, want exactly 1", marked)
	}

	due := WizardDue("1")
	marked = 0
	for _, row := range due.InlineKeyboard {
		for _, b := range row {
			if strings.HasPrefix(b.Text, "✓") {
				marked++
				if b.CallbackData == nil || *b.CallbackData != "nt_d_1" {
					t.Errorf("the marked due button is %v, want nt_d_1", b)
				}
			}
		}
	}
	if marked != 1 {
		t.Errorf("WizardDue(\"1\") marked %d buttons, want exactly 1", marked)
	}

	m := 30
	est := WizardEstimate(&m)
	marked = 0
	for _, row := range est.InlineKeyboard {
		for _, b := range row {
			if strings.HasPrefix(b.Text, "✓") {
				marked++
				if b.CallbackData == nil || *b.CallbackData != "nt_e_30" {
					t.Errorf("the marked estimate button is %v, want nt_e_30", b)
				}
			}
		}
	}
	if marked != 1 {
		t.Errorf("WizardEstimate(30) marked %d buttons, want exactly 1", marked)
	}
}

func TestWizardKeyboardsMarkNothingWhenNothingIsKnown(t *testing.T) {
	for name, kb := range map[string]tgbotapi.InlineKeyboardMarkup{
		"priority": WizardPriority(nil),
		"due":      WizardDue(""),
		"estimate": WizardEstimate(nil),
	} {
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if strings.HasPrefix(b.Text, "✓") {
					t.Errorf("%s: %q is marked as current with nothing known", name, b.Text)
				}
			}
		}
	}
}
