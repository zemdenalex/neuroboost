package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// «ℹ️ Что это?» (spec 21.09 §C). Denis: «на каждом пункте меню должна быть
// кнопка кроме отмены/возврата ещё и объяснение, потому что очень сложная
// система».
//
// The explanation arrives as its own message under the screen, and its «« Назад»
// deletes it. The screen itself is never touched, so «back» lands on exactly
// the screen that was open — with its task, its step and its caller — without
// this code having to know how to redraw any of them. Redrawing would have
// needed per-screen state: the priority picker alone has three callers, and a
// help screen that returned onboarding into Settings would be worse than none.

// Help screens. The callback is "help_" + screen; the text lives in
// handlers.helpText, one case per screen.
const (
	HelpToEvent  = "toevent"
	HelpToTask   = "totask"
	HelpLink     = "link"
	HelpRepeat   = "repeat"
	HelpNag      = "nag"
	HelpPostpone = "postpone"
	HelpPriority = "prio"
	HelpUpdates  = "updates"
	HelpQuick    = "quick"
)

// HelpScreens is every screen that has an explanation. A test asks for its
// text in both languages, so a screen added here without one goes red.
var HelpScreens = []string{
	HelpToEvent, HelpToTask, HelpLink, HelpRepeat, HelpNag, HelpPostpone, HelpPriority, HelpUpdates, HelpQuick,
}

// HelpButton sits next to a screen's cancel or back button.
func HelpButton(lang i18n.Lang, screen string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "ℹ️ Что это?", "ℹ️ What is this?"), "help_"+screen)
}

// HelpBack is under an explanation: it removes the explanation and nothing else.
func HelpBack(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "help_x")))
}
