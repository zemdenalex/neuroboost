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

// pressOn presses a button under a screen that has text, formatting and
// buttons — what Telegram hands the bot in cb.Message.
func pressOn(h *Handler, chat int64, data string, msg *tgbotapi.Message) {
	msg.Chat = &tgbotapi.Chat{ID: chat}
	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID: "cb", Data: data, From: &tgbotapi.User{ID: chat, FirstName: "Denis"}, Message: msg,
	})
}

func screenUnderHelp() *tgbotapi.Message {
	return &tgbotapi.Message{
		MessageID: 77,
		Text:      "Связать или перенести?",
		Entities:  []tgbotapi.MessageEntity{{Type: "bold", Offset: 0, Length: 7}},
		ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔗 Связать", "t2m_link")),
		}},
	}
}

// 🔴 Denis, 23.09 (pass 3, F7): «Смениться объяснением». The explanation
// REPLACES the screen it was pressed on, and «« Назад» puts that screen back
// exactly — text, bold and buttons — so the path goes on from the same step.
func TestHelpReplacesTheScreenAndBackRestoresIt(t *testing.T) {
	for _, screen := range keyboards.HelpScreens {
		h, fake, chat := quickHandler(t)
		pressOn(h, chat, "help_"+screen, screenUnderHelp())

		got := fake.last(t)
		if got.Method != "editMessageText" {
			t.Fatalf("%s: the explanation came as %s, want it in place of the screen", screen, got.Method)
		}
		if got.Text != helpText(i18n.RU, screen) || !strings.Contains(got.Markup, `"help_x"`) {
			t.Errorf("%s: %q / %s", screen, got.Text, got.Markup)
		}

		pressOn(h, chat, "help_x", &tgbotapi.Message{MessageID: 77, Text: got.Text})
		back := fake.last(t)
		if back.Method != "editMessageText" || back.Text != "Связать или перенести?" {
			t.Fatalf("%s: «Назад» gave %s %q, want the screen back", screen, back.Method, back.Text)
		}
		if !strings.Contains(back.Markup, `"t2m_link"`) || !strings.Contains(back.Entities, `"bold"`) {
			t.Errorf("%s: the screen came back without its buttons or bold: %s / %s", screen, back.Markup, back.Entities)
		}
		if len(fake.sent()) != 2 {
			t.Errorf("%s: help sent extra messages: %v", screen, fake.sent())
		}
	}
}

// An explanation posted by an earlier build (a separate message) has no screen
// saved behind it: its «« Назад» still deletes it, as it did.
func TestHelpBackOnAnOldExplanationDeletesIt(t *testing.T) {
	h, fake, chat := quickHandler(t)
	press(h, chat, "help_x")
	if !fake.called_("deleteMessage") {
		t.Fatalf("«« Назад» on an old explanation did not delete it; calls = %v", fake.calls())
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
