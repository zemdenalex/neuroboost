package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎯 Today"),
			tgbotapi.NewKeyboardButton("📋 Tasks"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Note"),
			tgbotapi.NewKeyboardButton("➕ New Task"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 New Event"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🗓 Calendar"),
			tgbotapi.NewKeyboardButton("📊 Stats"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Settings"),
		),
	)
}

func TaskPriority() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Emergency", "priority_1"),
			tgbotapi.NewInlineKeyboardButtonData("🟠 Urgent", "priority_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟡 Normal", "priority_3"),
			tgbotapi.NewInlineKeyboardButtonData("🟢 Low", "priority_4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚪ If Possible", "priority_5"),
			tgbotapi.NewInlineKeyboardButtonData("🔵 Buffer", "priority_0"),
		),
	)
}

func TaskActions(taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "task_done_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete", "task_delete_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "top_tasks"),
		),
	)
}

func BackToMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
		),
	)
}

// EventWhen offers the times people actually pick when they typed a title with
// no time. Deliberately four coarse choices, not a clock: this keyboard exists
// to rescue a half-typed line in one tap, and anything finer is faster to type.
func EventWhen() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сейчас", "when_now"),
			tgbotapi.NewInlineKeyboardButtonData("Через час", "when_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня вечером", "when_evening"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра утром", "when_tomorrow"),
		),
	)
}
