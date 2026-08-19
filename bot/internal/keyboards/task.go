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

// WizardPriority, WizardDue and WizardEstimate are the three steps behind
// "📝 Подробнее". Each carries wizardEscapes() — Denis, 18.08: nothing is
// required, every step must offer both "skip this field" and "save now".
func WizardPriority() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 1", "nt_p_1"),
			tgbotapi.NewInlineKeyboardButtonData("🟠 2", "nt_p_2"),
			tgbotapi.NewInlineKeyboardButtonData("🟡 3", "nt_p_3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 4", "nt_p_4"),
			tgbotapi.NewInlineKeyboardButtonData("⚪ 5", "nt_p_5"),
			tgbotapi.NewInlineKeyboardButtonData("🔵 0", "nt_p_0"),
		),
		wizardEscapes(),
	)
}

func WizardDue() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		dueRow("nt_d_"),
		wizardEscapes(),
	)
}

func WizardEstimate() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		estimateRow("nt_e_"),
		wizardEscapes(),
	)
}

// dueRow and estimateRow are the wizard's two data rows, factored out so the
// existing-task card (keyboards.go: TaskDue, TaskEstimate) offers exactly the
// same question and the same values under a different footer, instead of a
// second copy of the labels and offsets that drifts the first time one of
// them is edited. Only the callback prefix varies between callers.
func dueRow(prefix string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Сегодня", prefix+"0"),
		tgbotapi.NewInlineKeyboardButtonData("Завтра", prefix+"1"),
		tgbotapi.NewInlineKeyboardButtonData("Через неделю", prefix+"7"),
	)
}

func estimateRow(prefix string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("15м", prefix+"15"),
		tgbotapi.NewInlineKeyboardButtonData("30м", prefix+"30"),
		tgbotapi.NewInlineKeyboardButtonData("1ч", prefix+"60"),
		tgbotapi.NewInlineKeyboardButtonData("2ч", prefix+"120"),
	)
}

// wizardEscapes is the row that makes "nothing is required" true. It is one
// function so a new step cannot be added without it.
func wizardEscapes() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "nt_skip"),
		tgbotapi.NewInlineKeyboardButtonData("✅ Создать сейчас", "nt_save"),
	)
}
