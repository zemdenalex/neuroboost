package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainMenu is the reply keyboard, and it holds ENTRANCES ONLY.
//
// Denis, 18.08: six buttons — menu, calendar, events, tasks, create, settings —
// and "each of keyboard buttons open inline menus". The previous keyboard mixed
// entrances with actions (Note, New Task, New Event, Planning, Stats), and that
// is the mechanical reason inline screens kept saying "Use ➕ New Task" instead
// of offering a button: the action lived somewhere a screen could only point at.
func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Меню"),
			tgbotapi.NewKeyboardButton("🗓 Календарь"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 События"),
			tgbotapi.NewKeyboardButton("📋 Задачи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Создать"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)
}

// HomeInline is what 🏠 Меню actually offers — the screens that are no longer
// reply buttons.
func HomeInline() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Сегодня", "today_focus"),
			tgbotapi.NewInlineKeyboardButtonData("🗂 Планирование", "planning"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings_menu"),
		),
		// AMENDED BY CONTROLLER RULING — see the ledger.
		// Without this the "create_menu" route below has no emitter, and the home
		// screen offers no way to create anything, which contradicts the spec's
		// central rule that a screen carries the actions available from it.
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать", "create_menu"),
		),
	)
}

// CreateMenu is what ➕ Создать offers. 📝 Заметка still writes a task today;
// the notes entity is a separate spec
// (specs/2026-08-18-notes-as-their-own-entity-design.md) and this button moves
// to it unchanged when that lands.
func CreateMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Задача", "new_task"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Событие", "new_event"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Заметка", "new_note"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}
