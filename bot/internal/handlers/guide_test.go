package handlers

import (
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Denis, 17.09: Planning is hidden until free-slot placement exists.
func TestHomeMenuHasNoPlanningButton(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.RU, i18n.EN} {
		for _, row := range keyboards.HomeInline(lang).InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && *b.CallbackData == "planning" {
					t.Errorf("%s: the home menu still offers Planning", lang)
				}
			}
		}
	}
}

// The short guide opens by default, and 📖 swaps in the full one — in place,
// without ending the flow the user is writing into.
func TestShortGuideOpensTheFullOneInPlace(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewEventFlow(chat)

	short := fake.last(t)
	if strings.Count(short.Text, "\n") > 8 || !strings.Contains(short.Markup, "guide_full_event") {
		t.Fatalf("the event guide is not the short one with 📖: %q", short.Text)
	}

	press(h, chat, "guide_full_event")
	full := fake.last(t)
	// The fake answers edits with `true`, which the library cannot decode, so
	// editOrSend falls back to a send — the attempt is what shows «in place».
	if !fake.called_("editMessageText") || !strings.Contains(full.Text, "Ключевые слова") {
		t.Errorf("📖 did not edit in the full guide: %s %q", full.Method, full.Text)
	}
	if h.store.GetOrCreate(chat).CurrentFlow != "new_event" {
		t.Errorf("opening the full guide ended the flow")
	}
}
