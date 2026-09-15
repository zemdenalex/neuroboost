package keyboards

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
func DraftCard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", "dr_ok"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "dr_edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Отменить", "dr_cancel"),
		),
	)
}

// DraftFields are the fields the edit menu offers, declared once so the menu
// and the handler cannot drift apart.
//
// The callback data is short on purpose: Telegram caps callback_data at 64
// bytes, and this bot has already lost a feature to a callback that did not fit.
var DraftFields = []struct{ Label, Data string }{
	{"Название", "dre_title"},
	{"Дата", "dre_date"},
	{"Время", "dre_time"},
	{"Повтор", "dre_repeat"},
	{"Календарь", "dre_cal"},
	{"Цвет", "dre_colour"},
	{"Теги", "dre_tags"},
	{"Напоминания", "dre_remind"},
	{"Весь день", "dre_allday"},
}

func DraftEditMenu() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(DraftFields); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(DraftFields[i].Label, DraftFields[i].Data),
		}
		if i+1 < len(DraftFields) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(DraftFields[i+1].Label, DraftFields[i+1].Data))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// FreqPicker asks the question a bare «повтор» leaves open.
//
// «Без повтора» is here because the answer «actually, no repeat» must be
// reachable: the word may have been part of the title all along.
func FreqPicker() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Каждый день", "dr_freq_DAILY"),
			tgbotapi.NewInlineKeyboardButtonData("Каждую неделю", "dr_freq_WEEKLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Каждый месяц", "dr_freq_MONTHLY"),
			tgbotapi.NewInlineKeyboardButtonData("Каждый год", "dr_freq_YEARLY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Без повтора", "dr_freq_NONE"),
		),
	)
}

// ColourPicker offers the palette the web can actually paint.
var draftColours = []struct{ Label, Name string }{
	{"🔵 Синий", "blue"},
	{"🟣 Фиолетовый", "violet"},
	{"🟢 Зелёный", "green"},
	{"🔴 Красный", "red"},
	{"🟠 Янтарный", "amber"},
	{"🩵 Голубой", "cyan"},
	{"🩷 Розовый", "pink"},
	{"⚪ Серый", "slate"},
}

func ColourPicker() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(draftColours); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(draftColours[i].Label, "dr_col_"+draftColours[i].Name),
		}
		if i+1 < len(draftColours) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(draftColours[i+1].Label, "dr_col_"+draftColours[i+1].Name))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CalendarPicker lists the user's calendars by id.
func CalendarPicker(names, ids []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := range names {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(names[i], "dr_cal_"+ids[i]),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DraftDay offers the two days that cover most answers, with the text field
// still open for anything else.
func DraftDay() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "dr_day_0"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра", "dr_day_1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Послезавтра", "dr_day_2"),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
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
func ReminderPicker() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("За 10 минут", "dr_rem_10"),
			tgbotapi.NewInlineKeyboardButtonData("За 30 минут", "dr_rem_30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("За час", "dr_rem_60"),
			tgbotapi.NewInlineKeyboardButtonData("За день", "dr_rem_1440"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("По умолчанию", "dr_rem_default"),
			tgbotapi.NewInlineKeyboardButtonData("Не напоминать", "dr_rem_none"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
		),
	)
}

// DraftBack is what a text-input step shows: the user is expected to type, but
// a way back must stay on the screen. A prompt with no buttons at all is a
// dead end for anyone who changed their mind.
func DraftBack() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "dr_back"),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Отменить", "dr_cancel"),
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
func ListConfirm(n int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Одна запись", "dr_one"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Список из %d", n), "dr_many"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Отменить", "dr_cancel"),
		),
	)
}

// ListCard carries the same three answers as the single card. One card for the
// whole list, not one per entry: eight confirmations in a row is worse than not
// asking at all.
func ListCard(n int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Создать %d", n), "dr_makeall"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "dr_pick"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Отменить", "dr_cancel"),
		),
	)
}

// ListPick asks which entry to edit.
func ListPick(n int) tgbotapi.InlineKeyboardMarkup {
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
		tgbotapi.NewInlineKeyboardButtonData("⬅️ К списку", "dr_list"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DraftCardInList is the single card while a list is open: the same three
// buttons plus the way back to the list, which would otherwise be unreachable.
func DraftCardInList() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "dr_edit"),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ К списку", "dr_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Отменить всё", "dr_cancel"),
		),
	)
}
