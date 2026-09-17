package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// QuickAddKind is the question a line typed from nowhere gets: what is it?
//
// Its prefix, qa_, belongs to quick add alone. It is asked before the card's
// dr_ callbacks, and a shared prefix would route by declaration order.
func QuickAddKind(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Событие", "📅 Event"), "qa_event"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📋 Задача", "📋 Task"), "qa_task"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📝 Заметка", "📝 Note"), "qa_note"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "qa_cancel"),
		),
	)
}
