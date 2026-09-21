package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// The full «в календарь / сделать задачей» path (spec 21.09 §A2) — one set of
// buttons for both directions, told apart by prefix: "t2" (task → event) or
// "e2" (event → task).
//
// The choice lives in the flow, not in the button: a synthetic event id plus
// a step code does not always fit Telegram's 64 bytes, and a button that
// carries only its step cannot address the wrong task.

// ConvertCancel is the row every step ends with. Its «ℹ️» explains the
// direction; the link-or-move question has its own (convertCancelOn).
func ConvertCancel(lang i18n.Lang, prefix string) []tgbotapi.InlineKeyboardButton {
	screen := HelpToEvent
	if prefix == "e2" {
		screen = HelpToTask
	}
	return convertCancelOn(lang, prefix, screen)
}

func convertCancelOn(lang i18n.Lang, prefix, screen string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, screen),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), prefix+"x"),
	)
}

// ConvertHow asks link or move.
func ConvertHow(lang i18n.Lang, prefix string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔗 Связать", "🔗 Link"), prefix+"m_l"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➡️ Перенести", "➡️ Move"), prefix+"m_m"),
		),
		convertCancelOn(lang, prefix, HelpLink),
	)
}

// ConvertRepeat asks which part of a series. withOnce is false when nobody
// knows which day «this once» would be — an event opened as a whole series.
func ConvertRepeat(lang i18n.Lang, prefix string, withOnce bool) tgbotapi.InlineKeyboardMarkup {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔁 Вся серия", "🔁 Whole series"), prefix+"r_s"))
	if withOnce {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			i18n.T(lang, "1️⃣ Только этот раз", "1️⃣ Just this once"), prefix+"r_o"))
	}
	return tgbotapi.NewInlineKeyboardMarkup(row, ConvertCancel(lang, prefix))
}

// ToEventWhen offers the quick path's own slots — the same words for the same
// moments (TaskScheduleWhen) — plus a typed time.
func ToEventWhen(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сейчас", "Now"), "t2w_now"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Через час", "In an hour"), "t2w_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня вечером", "This evening"), "t2w_eve"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Завтра утром", "Tomorrow morning"), "t2w_tmr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Своё", "✏️ Custom"), "t2c"),
		),
		ConvertCancel(lang, "t2"),
	)
}

// ToEventDuration asks how long — only for a task with no estimate.
func ToEventDuration(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏱ 15м", "⏱ 15m"), "t2d_15"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏰ 30м", "⏰ 30m"), "t2d_30"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏰ 1ч", "⏰ 1h"), "t2d_60"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏰ 2ч", "⏰ 2h"), "t2d_120"),
		),
		ConvertCancel(lang, "t2"),
	)
}

// ConvertConfirm ends the path under the «what becomes what» card.
func ConvertConfirm(lang i18n.Lang, prefix string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Готово", "✅ Done"), prefix+"ok")),
		ConvertCancel(lang, prefix),
	)
}
