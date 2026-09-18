package keyboards

import (
	"fmt"
	"sort"
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
		{i18n.T(lang, "Конец повтора", "Repeat ends"), "dre_rend"},
		{i18n.T(lang, "Календарь", "Calendar"), "dre_cal"},
		{i18n.T(lang, "Цвет", "Colour"), "dre_colour"},
		{i18n.T(lang, "Теги", "Tags"), "dre_tags"},
		{i18n.T(lang, "Заметка", "Note"), "dre_note"},
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
			// «таблетки раз в 3 дня» — Denis, 16.09: «надо чтобы свои варианты были».
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Своя частота", "✏️ Custom"), "dr_freq_CUSTOM"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Без повтора", "No repeat"), "dr_freq_NONE"),
		),
	)
}

// RepeatEndPicker ends a series: never, or a count or a date typed as text.
// Not asked at creation — «lots of steps» — only offered under «Изменить».
func RepeatEndPicker(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Никогда", "Never"), "dr_rend_never"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ N раз или до даты", "✏️ N times or a date"), "dr_rend_text"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
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
// SpanFix is what a span typed back-to-front gets: swap the two dates, or
// write them again. 🔴 Not swapped silently — which of the two is the typo is
// the user's to say (Denis, 17.09).
func SpanFix(lang i18n.Lang, from, to string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf(i18n.T(lang, "🔄 Поменять: %s – %s", "🔄 Swap: %s – %s"), to, from), "dr_swap"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Написать даты", "✏️ Write the dates"), "dr_daytext"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "dr_back"),
		),
	)
}

func DraftDay(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня", "Today"), "dr_day_0"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Завтра", "Tomorrow"), "dr_day_1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Послезавтра", "In two days"), "dr_day_2"),
			// 🔴 A date the buttons do not cover — «14.10», «с 14.10 по 29.10»
			// — is typed. The button is what says that is allowed.
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Своя дата", "✏️ Another date"), "dr_daytext"),
		),
		tgbotapi.NewInlineKeyboardRow(
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
//
// 🔴 Ticks, not one answer (Denis, 16.09: «должны быть множественным выбором
// (кроме не напоминать — снимает все)… а также можно свое время»). A tap
// toggles and the picker stays open; «Готово» returns to the card. A typed time
// joins the offered ones as a button of its own, so it can be unticked too.
func ReminderPicker(lang i18n.Lang, offsets *[]int) tgbotapi.InlineKeyboardMarkup {
	chosen := map[int]bool{}
	shown := []int{10, 30, 60, 1440}
	if offsets != nil {
		for _, m := range *offsets {
			chosen[m] = true
			if !containsOffset(shown, m) {
				shown = append(shown, m)
			}
		}
	}
	sort.Ints(shown)

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, m := range shown {
		label := offsetLabel(lang, m)
		if chosen[m] {
			label = "✅ " + label
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, "dr_rem_"+strconv.Itoa(m)))
		if len(row) == 2 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	mark := func(on bool, label string) string {
		if on {
			return "✅ " + label
		}
		return label
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Своё время", "✏️ Own time"), "dr_rem_custom"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark(offsets == nil, i18n.T(lang, "По умолчанию", "My default")), "dr_rem_default"),
			tgbotapi.NewInlineKeyboardButtonData(mark(offsets != nil && len(*offsets) == 0, i18n.T(lang, "Не напоминать", "No reminder")), "dr_rem_none"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✔️ Готово", "✔️ Done"), "dr_rem_done"),
		),
	)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func containsOffset(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// offsetLabel says an offset the way the picker always has: «За час».
func offsetLabel(lang i18n.Lang, m int) string {
	switch {
	case m == 60:
		return i18n.T(lang, "За час", "An hour before")
	case m == 1440:
		return i18n.T(lang, "За день", "A day before")
	case m%1440 == 0:
		return fmt.Sprintf(i18n.T(lang, "За %d дн.", "%d days before"), m/1440)
	case m%60 == 0:
		return fmt.Sprintf(i18n.T(lang, "За %d ч.", "%d hours before"), m/60)
	default:
		return fmt.Sprintf(i18n.T(lang, "За %d мин.", "%d minutes before"), m)
	}
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

// TaskListItem is one task picked from the list: rewrite it, drop it, or go
// back. dr_trew_/dr_tdel_ are distinct from every other dr_ prefix — no one of
// them is a prefix of another.
func TaskListItem(lang i18n.Lang, i int) tgbotapi.InlineKeyboardMarkup {
	n := strconv.Itoa(i)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Переписать", "✏️ Rewrite"), "dr_trew_"+n),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Убрать из списка", "🗑 Remove from list"), "dr_tdel_"+n),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ К списку", "⬅️ Back to list"), "dr_list"),
		),
	)
}

// ManyDates is what a line with two or more dates gets: one event across them
// all, or one per date.
//
// 🔴 Asked, never guessed (Denis, 17.09: «если просто 2 даты написано, то он
// должен спросить… и дать варианты»).
func ManyDates(lang i18n.Lang, from, to string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf(i18n.T(lang, "📆 Одно событие: %s – %s", "📆 One event: %s – %s"), from, to), "dr_dspan"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "☑️ Выбрать даты", "☑️ Pick the dates"), "dr_dpick"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// DatePicker ticks the dates an event should be created on. Every date starts
// ticked: the user wrote them all.
func DatePicker(lang i18n.Lang, labels []string, chosen []bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, label := range labels {
		if i < len(chosen) && chosen[i] {
			label = "✅ " + label
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, "dr_dtog_"+strconv.Itoa(i)))
		if len(row) == 3 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Создать выбранные", "✅ Create the ticked ones"), "dr_dmake"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
