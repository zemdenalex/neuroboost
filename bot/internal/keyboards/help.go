package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// «ℹ️ Что это?» (spec 21.09 §C). Denis: «на каждом пункте меню должна быть
// кнопка кроме отмены/возврата ещё и объяснение, потому что очень сложная
// система».
//
// Since 23.09 (Denis, pass 3, F7: «Смениться объяснением») the explanation
// takes the screen's place and «« Назад» puts the screen back. No screen has
// to know how to redraw itself: the handler saves the pressed message as
// Telegram showed it — text, formatting, buttons — and restores that
// (handlers/help.go). Before, the explanation came as a separate message,
// precisely to avoid per-screen redraw state; the snapshot removes the need.

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
	HelpDayTasks = "daytasks"
)

// HelpScreens is every screen that has an explanation. A test asks for its
// text in both languages, so a screen added here without one goes red.
var HelpScreens = []string{
	HelpToEvent, HelpToTask, HelpLink, HelpRepeat, HelpNag, HelpPostpone, HelpPriority, HelpUpdates, HelpQuick, HelpDayTasks,
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
