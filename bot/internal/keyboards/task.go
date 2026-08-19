package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// TaskCard is what a parsed line offers.
//
// 🔴 ✅ Создать is FIRST and always enabled. Denis, 18.08: "all of them
// shouldn't be required to create the task". The wizard is an offer underneath
// it, never a gate in front of it.
func TaskCard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Создать", "nt_save"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Подробнее (по шагам)", "nt_wizard"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "main_menu"),
		),
	)
}
