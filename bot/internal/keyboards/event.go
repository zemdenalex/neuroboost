package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Picking and editing an event.
//
// Denis, 16.09: «нет возможности редактировать событие, задачи можно выбрать и
// редактировать, а события нет». The agenda printed a list of lines nobody
// could touch — the events existed on screen and did not exist as controls.

// EventPageSize is how many events one page of the picker shows.
//
// 🔴 A page, not a cap. Twelve used to be the limit and the thirteenth event
// was unreachable (Denis, 17.09: «Не могу открыть события позже первых 12, надо
// сделать первые 10… кнопки влево, номер страницы, вправо»). Ten plus the pager
// stays well under what Telegram renders — an over-long keyboard is rejected
// whole.
const EventPageSize = 10

// EventPicker turns a list of events into buttons.
//
// ⚠ Titles are the user's own text and are not translated or trimmed of
// meaning — only of length, because a button label longer than the screen
// helps nobody.
//
// page is zero-based and clamped. The pager row — ◀ · page/total · ▶ — sits
// above «Назад» and appears only when there is more than one page; the number
// itself returns to the first page.
func EventPicker(lang i18n.Lang, labels, ids []string, back string, page int) tgbotapi.InlineKeyboardMarkup {
	pages := (len(labels) + EventPageSize - 1) / EventPageSize
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	from := page * EventPageSize
	for i := from; i < len(labels) && i < from+EventPageSize; i++ {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(labels[i], "ev_"+ids[i]),
		})
	}
	if pages > 1 {
		prev, next := page-1, page+1
		if prev < 0 {
			prev = pages - 1
		}
		if next >= pages {
			next = 0
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("◀", fmt.Sprintf("evp_%d", prev)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page+1, pages), "evp_0"),
			tgbotapi.NewInlineKeyboardButtonData("▶", fmt.Sprintf("evp_%d", next)),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), back),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// EventEditor is an opened event: every field one tap away, save and delete on
// the same screen.
//
// 🔴 Denis, 17.09: «События — изменить — выбор события — изменить — изменить —
// выбор характеристики, если мы и так хотим изменить событие, то зачем еще два
// раза это подтверждать». Opening an event from «✏️ Изменить» IS the intent to
// change it, so the fields are the first screen, not the third.
func EventEditor(lang i18n.Lang, id string) tgbotapi.InlineKeyboardMarkup {
	fields := draftFields(lang)
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(fields); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(fields[i].Label, fields[i].Data)}
		if i+1 < len(fields) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(fields[i+1].Label, fields[i+1].Data))
		}
		rows = append(rows, row)
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "💾 Сохранить", "💾 Save"), "dr_ok"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Удалить", "🗑 Delete"), "evd_"+id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Не сохранять", "⬅️ Discard"), "event_pick"),
		),
	)
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
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "ev_"+id),
		),
	)
}
