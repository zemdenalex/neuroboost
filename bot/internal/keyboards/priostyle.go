package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// PriorityStyle picks the priority symbol (spec 21.09 §B) — one screen for
// Settings, onboarding and the one-time question, told apart by prefix.
// Each button carries its own preview so the choice needs no reading.
func PriorityStyle(lang i18n.Lang, current, prefix, backData, backLabel string) tgbotapi.InlineKeyboardMarkup {
	btn := func(style, label string) []tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(tick(style == current)+label, prefix+style))
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		btn("circles", i18n.T(lang, "🔴 🟠 🟡  кружки", "🔴 🟠 🟡  circles")),
		btn("dot", i18n.T(lang, "●1 ●2 ○3  точки", "●1 ●2 ○3  dots")),
		btn("dash", i18n.T(lang, "— — —  тире", "— — —  dashes")),
		tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpPriority), tgbotapi.NewInlineKeyboardButtonData(backLabel, backData)),
	)
}
