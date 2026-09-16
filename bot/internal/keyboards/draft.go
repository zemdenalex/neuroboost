package keyboards

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// The confirmation card, Denis's design of 15.09: «в подтверждении должно быть
// 3 кнопки подтвердить, изменить, удалить/отменить и если изменить то уже
// выбирается изменить что».
//
// 🔴 The card is shown for EVERY creation, including a line the bot understood
// completely. That is the point: the bot stops writing into his calendar things
// he has not seen. It also gives the clarifications — frequency, missing date,
// reminder preset — one place to live instead of three ad-hoc question states
// colliding in FlowStep.
func DraftCard(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Подтвердить", "✅ Confirm"), "dr_ok"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "dr_edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// draftFields are the fields the edit menu offers, declared once so the menu
// and the handler cannot drift apart.
//
// The callback data is short on purpose: Telegram caps callback_data at 64
// bytes, and this bot has already lost a feature to a callback that did not fit.
//
// ⚠ It became a function when the labels became translatable. The callback
// data did NOT: a translated callback is an address nobody can route.
func draftFields(lang i18n.Lang) []struct{ Label, Data string } {
	return []struct{ Label, Data string }{
		{i18n.T(lang, "Название", "Title"), "dre_title"},
		{i18n.T(lang, "Дата", "Date"), "dre_date"},
		{i18n.T(lang, "Время", "Time"), "dre_time"},
		{i18n.T(lang, "Повтор", "Repeat"), "dre_repeat"},
		{i18n.T(lang, "Календарь", "Calendar"), "dre_cal"},
		{i18n.T(lang, "Цвет", "Colour"), "dre_colour"},
		{i18n.T(lang, "Теги", "Tags"), "dre_tags"},
		{i18n.T(lang, "Напоминания", "Reminders"), "dre_remind"},
		{i18n.T(lang, "Весь день", "All day"), "dre_allday"},
	}
}

func DraftEditMenu(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	fields := draftFields(lang)
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(fields); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(fields[i].Label, fields[i].Data),
		}
		if i+1 < len(fields) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(fields[i+1].Label, fields[i+1].Data))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// FreqPicker asks the question a bare «повтор» leaves open.
//
// «Без повтора» is here because the answer «actually, no repeat» must be
// reachable: the word may have been part of the title all along.
func FreqPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый день", "Every day"), "dr_freq_DAILY"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждую неделю", "Every week"), "dr_freq_WEEKLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый месяц", "Every month"), "dr_freq_MONTHLY"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Каждый год", "Every year"), "dr_freq_YEARLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Без повтора", "No repeat"), "dr_freq_NONE"),
		),
	)
}

// draftColours offers the palette the web can actually paint.
func draftColours(lang i18n.Lang) []struct{ Label, Name string } {
	return []struct{ Label, Name string }{
		{i18n.T(lang, "🔵 Синий", "🔵 Blue"), "blue"},
		{i18n.T(lang, "🟣 Фиолетовый", "🟣 Violet"), "violet"},
		{i18n.T(lang, "🟢 Зелёный", "🟢 Green"), "green"},
		{i18n.T(lang, "🔴 Красный", "🔴 Red"), "red"},
		{i18n.T(lang, "🟠 Янтарный", "🟠 Amber"), "amber"},
		{i18n.T(lang, "🩵 Голубой", "🩵 Cyan"), "cyan"},
		{i18n.T(lang, "🩷 Розовый", "🩷 Pink"), "pink"},
		{i18n.T(lang, "⚪ Серый", "⚪ Slate"), "slate"},
	}
}

func ColourPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	colours := draftColours(lang)
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(colours); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(colours[i].Label, "dr_col_"+colours[i].Name),
		}
		if i+1 < len(colours) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(colours[i+1].Label, "dr_col_"+colours[i+1].Name))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CalendarPicker lists the user's calendars by id.
//
// ⚠ The names are the user's own and are NOT translated — a calendar called
// «Работа» is called «Работа» in every interface language, the way a person's
// name is.
func CalendarPicker(lang i18n.Lang, names, ids []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := range names {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(names[i], "dr_cal_"+ids[i]),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DraftDay offers the days that cover most answers, with the text field still
// open for anything else.
func DraftDay(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня", "Today"), "dr_day_0"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Завтра", "Tomorrow"), "dr_day_1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Послезавтра", "In two days"), "dr_day_2"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
		),
	)
}

// ReminderPicker offers the common offsets plus the two answers that are not
// offsets at all.
//
// 🔴 «По умолчанию» and «Не напоминать» are DIFFERENT, and the difference is
// invisible unless both are on the keyboard: the first leaves the field absent
// so the server applies the user's preset, the second writes an empty list and
// means silence forever.
func ReminderPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "За 10 минут", "10 minutes before"), "dr_rem_10"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "За 30 минут", "30 minutes before"), "dr_rem_30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "За час", "An hour before"), "dr_rem_60"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "За день", "A day before"), "dr_rem_1440"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "По умолчанию", "My default"), "dr_rem_default"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Не напоминать", "No reminder"), "dr_rem_none"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
		),
	)
}

// DraftBack is what a text-input step shows: the user is expected to type, but
// a way back must stay on the screen. A prompt with no buttons at all is a
// dead end for anyone who changed their mind.
func DraftBack(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// ListConfirm is the question Denis asked for by name: «это все должно
// уточняться создать одну задачу с таким длинным описанием/названием или это
// список задач».
//
// 🔴 The bot asks and never decides. Five tasks glued into one title and one
// long title split into five tasks are equally wrong, and only the person who
// typed it knows which they meant.
func ListConfirm(lang i18n.Lang, n int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Одна запись", "One entry"), "dr_one"),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf(i18n.T(lang, "Список из %d", "A list of %d"), n), "dr_many"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// ListCard carries the same three answers as the single card. One card for the
// whole list, not one per entry: eight confirmations in a row is worse than not
// asking at all.
func ListCard(lang i18n.Lang, n int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf(i18n.T(lang, "✅ Создать %d", "✅ Create %d"), n), "dr_makeall"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "dr_pick"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// ListPick asks which entry to edit.
func ListPick(lang i18n.Lang, n int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i := 0; i < n; i++ {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			strconv.Itoa(i+1), "dr_item_"+strconv.Itoa(i)))
		if len(row) == 5 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ К списку", "⬅️ Back to list"), "dr_list"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DraftCardInList is the single card while a list is open: the same buttons
// plus the way back to the list, which would otherwise be unreachable.
func DraftCardInList(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "dr_edit"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ К списку", "⬅️ Back to list"), "dr_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить всё", "🗑 Cancel all"), "dr_cancel"),
		),
	)
}
