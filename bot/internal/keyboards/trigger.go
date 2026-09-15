package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Keyboards for ⚙️ Настройки → 🔤 Ключевые слова.
//
// 🔴 Their callbacks carry their own prefixes (`kwf_`, `kwv_`) rather than
// reusing the confirmation card's `dr_col_` / `dr_freq_`. Reusing them would
// have been three fewer functions and one real defect: the card's handler
// refuses every `dr_` callback that arrives outside the creation flow, so a
// colour chosen here would have answered «это создание уже закрыто».

// TriggerFieldPicker asks what a word stands for.
//
// The labels are passed in already translated: they come from
// parse.TriggerFields, and this package of pure rendering must not depend on
// the parser.
func TriggerFieldPicker(lang i18n.Lang, labels, names []string) tgbotapi.InlineKeyboardMarkup {
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
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "settings_keywords"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// TriggerColourPicker offers the palette for a word that means a colour.
func TriggerColourPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	colours := draftColours(lang)
	for i := 0; i < len(colours); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(colours[i].Label, "kwv_col_"+colours[i].Name),
		}
		if i+1 < len(colours) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(colours[i+1].Label, "kwv_col_"+colours[i+1].Name))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "settings_keywords"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// TriggerFreqPicker offers the frequencies for a word that means a repeat.
func TriggerFreqPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый день", "Every day"), "kwv_freq_DAILY"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждую неделю", "Every week"), "kwv_freq_WEEKLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый месяц", "Every month"), "kwv_freq_MONTHLY"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый год", "Every year"), "kwv_freq_YEARLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "settings_keywords"),
		),
	)
}

// TriggerCancel is what a text-input step shows.
func TriggerCancel(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "settings_keywords"),
		),
	)
}
