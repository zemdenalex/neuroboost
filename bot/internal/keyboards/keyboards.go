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
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать", "task_sched_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "task_done_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete", "task_delete_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "top_tasks"),
		),
	)
}

// Scheduling a task carries its whole state in the callback data — task, slot,
// then duration — and touches the session store not at all.
//
// The session would have been the shorter road: new_event already keeps a
// half-built event in FlowData. But a task card is a message that stays in the
// chat. A user with two cards open, or one who taps a card from yesterday,
// would schedule whatever the session happened to hold last, and a bot restart
// (this one is redeployed by hand, gotcha 19) would silently empty it. Packing
// instead makes every button self-describing and idempotent.
//
// The budget is Telegram's 64 bytes of callback_data. A UUID is 36 of them,
// which leaves 28 — enough for these prefixes and not much else, so
// keyboards_test.go asserts the limit rather than trusting the arithmetic here.

// TaskScheduleWhen asks which day and hour, in the four shapes people pick.
func TaskScheduleWhen(taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сейчас", "task_when_"+taskID+"_now"),
			tgbotapi.NewInlineKeyboardButtonData("Через час", "task_when_"+taskID+"_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня вечером", "task_when_"+taskID+"_eve"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра утром", "task_when_"+taskID+"_tmr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Отмена", "task_action_"+taskID),
		),
	)
}

// TaskScheduleDuration asks how long, carrying the slot forward so the answer
// needs nothing remembered.
func TaskScheduleDuration(taskID, slot string) tgbotapi.InlineKeyboardMarkup {
	at := func(minutes string) tgbotapi.InlineKeyboardButton {
		label := map[string]string{"15": "⏱ 15м", "30": "⏰ 30м", "60": "⏰ 1ч", "120": "⏰ 2ч"}[minutes]
		return tgbotapi.NewInlineKeyboardButtonData(label, "task_plan_"+taskID+"_"+slot+"_"+minutes)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(at("15"), at("30")),
		tgbotapi.NewInlineKeyboardRow(at("60"), at("120")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "task_sched_"+taskID),
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
