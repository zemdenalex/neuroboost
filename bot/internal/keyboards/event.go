package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Picking and editing an event.
//
// Denis, 16.09: «нет возможности редактировать событие, задачи можно выбрать и
// редактировать, а события нет». The agenda printed a list of lines nobody
// could touch — the events existed on screen and did not exist as controls.

// EventPickLimit caps how many events become buttons on one screen.
//
// 🔴 The agenda covers two weeks, and a busy fortnight is more rows than
// Telegram will render — an over-long keyboard is REJECTED WHOLE, so the screen
// fails rather than the extra rows dropping off. The list text still shows
// everything; only the buttons are capped, and the screen says so.
const EventPickLimit = 12

// EventPicker turns a list of events into buttons.
//
// ⚠ Titles are the user's own text and are not translated or trimmed of
// meaning — only of length, because a button label longer than the screen
// helps nobody.
func EventPicker(lang i18n.Lang, labels, ids []string, back string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := range labels {
		if i >= EventPickLimit {
			break
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(labels[i], "ev_"+ids[i]),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), back),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// EventCard is one event, with what can be done to it.
func EventCard(lang i18n.Lang, id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "eve_"+id),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Удалить", "🗑 Delete"), "evd_"+id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 События", "📅 Events"), "agenda_open"),
		),
	)
}

// EventDeleteConfirm asks once before deleting.
//
// 🔴 A delete button that acts on the first press is a delete button pressed by
// accident. This is the only destructive action the bot offers on an event, and
// the event may be a whole repeating series.
func EventDeleteConfirm(lang i18n.Lang, id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Да, удалить", "🗑 Yes, delete"), "evdy_"+id),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Отмена", "⬅️ Cancel"), "ev_"+id),
		),
	)
}

// DraftCardEditing is the confirmation card for an event that already exists.
//
// ⚠ «Сохранить», not «Подтвердить» — and there is no ✅ Создать anywhere on it.
// The words are the only thing telling the user whether they are about to add a
// second event or change the one they opened.
func DraftCardEditing(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "💾 Сохранить", "💾 Save"), "dr_ok"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "dr_edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Не сохранять", "⬅️ Discard"), "dr_cancel"),
		),
	)
}
