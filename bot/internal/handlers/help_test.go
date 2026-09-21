package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Every screen with a «ℹ️ Что это?» has its explanation in both languages, and
// the two are not the same string — a missing translation falls back silently.
func TestEveryHelpScreenHasItsTextInBothLanguages(t *testing.T) {
	for _, screen := range keyboards.HelpScreens {
		ru, en := helpText(i18n.RU, screen), helpText(i18n.EN, screen)
		if ru == "" || en == "" {
			t.Errorf("%s: no explanation (ru %d chars, en %d chars)", screen, len(ru), len(en))
			continue
		}
		if ru == en {
			t.Errorf("%s: the Russian and English explanations are the same string", screen)
		}
	}
}

// The press answers with the explanation as a NEW message under the screen —
// the screen is not edited, so «« Назад» has nothing to redraw.
func TestHelpPressSendsTheExplanationAndLeavesTheScreen(t *testing.T) {
	for _, screen := range keyboards.HelpScreens {
		h, fake, chat := quickHandler(t)
		press(h, chat, "help_"+screen)

		got := fake.last(t)
		if got.Method != "sendMessage" {
			t.Errorf("%s: the explanation came as %s — the screen under it was overwritten", screen, got.Method)
		}
		if got.Text != helpText(i18n.RU, screen) {
			t.Errorf("%s: sent %q, not the registry's text", screen, got.Text)
		}
		if !strings.Contains(got.Markup, `"help_x"`) {
			t.Errorf("%s: the explanation has no «« Назад»: %s", screen, got.Markup)
		}
		if fake.called_("editMessageText") {
			t.Errorf("%s: a help press edited a message", screen)
		}
	}
}

func TestHelpBackDeletesOnlyTheExplanation(t *testing.T) {
	h, fake, chat := quickHandler(t)
	press(h, chat, "help_x")
	if !fake.called_("deleteMessage") {
		t.Fatalf("«« Назад» did not delete the explanation; calls = %v", fake.calls())
	}
	if len(fake.sent()) != 0 {
		t.Errorf("«« Назад» also sent something: %v", fake.sent())
	}
}

// A help press in the middle of the convert path must not end it: the
// explanation is read and the path goes on from the same step.
func TestHelpDoesNotEndTheFlowItIsPressedIn(t *testing.T) {
	h, _, chat := quickHandler(t)
	us := h.store.GetOrCreate(chat)
	us.CurrentFlow, us.FlowStep = toEventFlow, "time_text"
	press(h, chat, "help_"+keyboards.HelpToEvent)
	if us.CurrentFlow != toEventFlow || us.FlowStep != "time_text" {
		t.Errorf("help ended the flow: now %q / %q", us.CurrentFlow, us.FlowStep)
	}
}

// An unknown screen — a button from a later build, or a typo — answers
// honestly rather than with an empty message Telegram would refuse.
func TestHelpForAnUnknownScreenSaysSo(t *testing.T) {
	h, fake, chat := quickHandler(t)
	press(h, chat, "help_nosuchscreen")
	got := fake.last(t)
	if !strings.Contains(got.Text, "устарел") {
		t.Errorf("unknown screen answered %q", got.Text)
	}
}

// Each screen spec §C2 names carries its own button — the registry having a
// text is not the same as the user being able to reach it.
func TestTheC2ScreensCarryTheirHelpButton(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	cases := map[string]string{
		"TaskRepeat":           markupOf(keyboards.TaskRepeat(i18n.RU, id, "")),
		"TaskNag":              markupOf(keyboards.TaskNag(i18n.RU, id, 0)),
		"TaskPostpone":         markupOf(keyboards.TaskPostpone(i18n.RU, id)),
		"PriorityStyle":        markupOf(keyboards.PriorityStyle(i18n.RU, "circles", "prs_", "settings_menu", "«")),
		"UpdatesToggle":        markupOf(keyboards.UpdatesToggle(i18n.RU, true)),
		"UpdatesOff":           markupOf(keyboards.UpdatesOff(i18n.RU)),
		"ConvertHow t2":        markupOf(keyboards.ConvertHow(i18n.RU, "t2")),
		"ConvertHow e2":        markupOf(keyboards.ConvertHow(i18n.RU, "e2")),
		"ToEventWhen":          markupOf(keyboards.ToEventWhen(i18n.RU)),
		"ConvertConfirm e2":    markupOf(keyboards.ConvertConfirm(i18n.RU, "e2")),
		"ConvertRepeat t2":     markupOf(keyboards.ConvertRepeat(i18n.RU, "t2", true)),
		"ToEventDuration":      markupOf(keyboards.ToEventDuration(i18n.RU)),
		"ConvertConfirm t2":    markupOf(keyboards.ConvertConfirm(i18n.RU, "t2")),
		"ConvertRepeat e2 all": markupOf(keyboards.ConvertRepeat(i18n.RU, "e2", false)),
	}
	want := map[string]string{
		"TaskRepeat": keyboards.HelpRepeat, "TaskNag": keyboards.HelpNag, "TaskPostpone": keyboards.HelpPostpone,
		"PriorityStyle": keyboards.HelpPriority, "UpdatesToggle": keyboards.HelpUpdates, "UpdatesOff": keyboards.HelpUpdates,
		"ConvertHow t2": keyboards.HelpLink, "ConvertHow e2": keyboards.HelpLink,
		"ToEventWhen": keyboards.HelpToEvent, "ToEventDuration": keyboards.HelpToEvent,
		"ConvertConfirm t2": keyboards.HelpToEvent, "ConvertRepeat t2": keyboards.HelpToEvent,
		"ConvertConfirm e2": keyboards.HelpToTask, "ConvertRepeat e2 all": keyboards.HelpToTask,
	}
	for name, markup := range cases {
		if !strings.Contains(markup, `"help_`+want[name]+`"`) {
			t.Errorf("%s has no «ℹ️ Что это?» for %q: %s", name, want[name], markup)
		}
	}
}

// markupOf is a keyboard as Telegram would receive it.
func markupOf(kb tgbotapi.InlineKeyboardMarkup) string {
	b, err := json.Marshal(kb)
	if err != nil {
		return ""
	}
	return string(b)
}

// The old per-flow «ℹ️» (t2i/e2i) is gone: one registry, one renderer.
func TestConvertNoLongerHasItsOwnHelp(t *testing.T) {
	for _, m := range []string{markupOf(keyboards.ConvertHow(i18n.RU, "t2")), markupOf(keyboards.ConvertHow(i18n.RU, "e2"))} {
		if strings.Contains(m, `"t2i"`) || strings.Contains(m, `"e2i"`) {
			t.Errorf("the convert path still carries its own help callback: %s", m)
		}
	}
}
