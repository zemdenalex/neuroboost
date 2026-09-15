package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// Keyboards for ⚙️ Настройки → 🔤 Ключевые слова.
//
// 🔴 Their callbacks carry their own prefixes (`kwf_`, `kwv_`) rather than
// reusing the confirmation card's `dr_col_` / `dr_freq_`. Reusing them would
// have been three fewer functions and one real defect: the card's handler
// refuses every `dr_` callback that arrives outside the creation flow, so a
// colour chosen here would have answered «это создание уже закрыто».

// TriggerFieldPicker asks what a word stands for.
//
// The labels come from parse.TriggerFields, but this package cannot import
// parse (parse would then import keyboards through nothing, but the dependency
// still points the wrong way for a package of pure rendering), so the caller
// passes them in — already paired with the stored names.
func TriggerFieldPicker(labels, names []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(labels); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(labels[i], "kwf_"+names[i]),
		}
		if i+1 < len(labels) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(labels[i+1], "kwf_"+names[i+1]))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "settings_keywords"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// TriggerColourPicker offers the palette for a word that means a colour.
func TriggerColourPicker() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(draftColours); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(draftColours[i].Label, "kwv_col_"+draftColours[i].Name),
		}
		if i+1 < len(draftColours) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(draftColours[i+1].Label, "kwv_col_"+draftColours[i+1].Name))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "settings_keywords"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// TriggerFreqPicker offers the frequencies for a word that means a repeat.
func TriggerFreqPicker() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Каждый день", "kwv_freq_DAILY"),
			tgbotapi.NewInlineKeyboardButtonData("Каждую неделю", "kwv_freq_WEEKLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Каждый месяц", "kwv_freq_MONTHLY"),
			tgbotapi.NewInlineKeyboardButtonData("Каждый год", "kwv_freq_YEARLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "settings_keywords"),
		),
	)
}

// TriggerCancel is what a text-input step shows.
func TriggerCancel() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "settings_keywords"),
		),
	)
}
