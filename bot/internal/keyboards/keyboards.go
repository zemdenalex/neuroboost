package keyboards

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// TaskActions is the card for one existing task.
//
// Срок / Оценка / Теги reuse the wizard's own step screens rather than
// duplicating them: the question "when is this due" has one right keyboard, and
// two copies of it drift apart the first time one is edited.
//
// Budget check, not arithmetic in prose: keyboards_test.go asserts every
// callback_data here against Telegram's 64 bytes. task_sched_ + a 36-character
// UUID is the longest button ON THIS CARD, at 47; the screens these three
// buttons open (TaskDue, TaskEstimate below) carry a longer "_set_" callback
// of their own and are asserted the same way.
//
// ⚠ Translating the LABELS changes none of that arithmetic: the budget is on
// callback_data, which is never translated.
func TaskActions(lang i18n.Lang, taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏰ Запланировать", "⏰ Schedule"), "task_sched_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Срок", "📅 Due"), "task_due_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏱ Оценка", "⏱ Estimate"), "task_est_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏷 Теги", "🏷 Tags"), "task_tag_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Готово", "✅ Done"), "task_done_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Удалить", "🗑 Delete"), "task_delete_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "top_tasks"),
		),
	)
}

// TaskDue and TaskEstimate answer "📅 Срок" / "⏱ Оценка" on an existing task's
// card. They reuse dueRow / estimateRow (task.go) — the wizard's own two data
// rows — so the values offered here can never drift from what "📝 Подробнее"
// offers when creating a task.
//
// The footer is the one thing that could not be reused as-is: wizardEscapes()
// carries "⏭ Пропустить" / "✅ Создать сейчас", both worded for the moment a
// task is being created. Firing "✅ Создать сейчас" on a task that already
// exists would be a button that does the wrong thing while claiming to create
// one — worse than a dead button, not better — so this footer is its own row,
// a plain cancel back to the card.
//
// The callback prefix also can't be nt_d_/nt_e_: those carry no task id, and
// this bot deliberately keeps every task button self-describing (see
// TaskScheduleWhen's comment below) rather than remembering which task is
// "current" in session state, which breaks the moment two cards are open. So
// the value-setting buttons here get their own "task_due_set_"/"task_est_set_"
// prefix, one level more specific than the card's own "task_due_"/"task_est_"
// — HandleCallback must check the "_set_" prefix first, same rule as every
// other prefix switch in this bot.
func TaskDue(lang i18n.Lang, taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		dueRow(lang, "task_due_set_"+taskID+"_", ""),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Отмена", "« Cancel"), "task_action_"+taskID),
		),
	)
}

func TaskEstimate(lang i18n.Lang, taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		estimateRow(lang, "task_est_set_"+taskID+"_", ""),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Отмена", "« Cancel"), "task_action_"+taskID),
		),
	)
}

// Scheduling a task carries its whole state in the callback data — task, slot,
// then duration — and touches the session store not at all.
//
// The session would have been the shorter road: new_event already keeps a
// half-built event in FlowData. But a task card is a message that stays in the
// chat. A user with two cards open, or one who taps a card from yesterday,
// would schedule whatever the session happened to hold last, and a bot restart
// (this one is redeployed by hand, gotcha 19) would silently empty it. Packing
// instead makes every button self-describing and idempotent.
//
// The budget is Telegram's 64 bytes of callback_data. A UUID is 36 of them,
// which leaves 28 — enough for these prefixes and not much else, so
// keyboards_test.go asserts the limit rather than trusting the arithmetic here.

// TaskScheduleWhen asks which day and hour, in the four shapes people pick.
func TaskScheduleWhen(lang i18n.Lang, taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сейчас", "Now"), "task_when_"+taskID+"_now"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Через час", "In an hour"), "task_when_"+taskID+"_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня вечером", "This evening"), "task_when_"+taskID+"_eve"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Завтра утром", "Tomorrow morning"), "task_when_"+taskID+"_tmr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Отмена", "« Cancel"), "task_action_"+taskID),
		),
	)
}

// TaskScheduleDuration asks how long, carrying the slot forward so the answer
// needs nothing remembered.
func TaskScheduleDuration(lang i18n.Lang, taskID, slot string) tgbotapi.InlineKeyboardMarkup {
	at := func(minutes string) tgbotapi.InlineKeyboardButton {
		label := map[string]string{
			"15":  i18n.T(lang, "⏱ 15м", "⏱ 15m"),
			"30":  i18n.T(lang, "⏰ 30м", "⏰ 30m"),
			"60":  i18n.T(lang, "⏰ 1ч", "⏰ 1h"),
			"120": i18n.T(lang, "⏰ 2ч", "⏰ 2h"),
		}[minutes]
		return tgbotapi.NewInlineKeyboardButtonData(label, "task_plan_"+taskID+"_"+slot+"_"+minutes)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(at("15"), at("30")),
		tgbotapi.NewInlineKeyboardRow(at("60"), at("120")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "task_sched_"+taskID),
		),
	)
}

func BackToMenu(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu"),
		),
	)
}

// BackToTasks is the recovery keyboard for a callback the bot could not parse
// — a real button back to the task list, not prose that names one which the
// user may no longer have on screen (or, once, does not exist at all: this
// replaced text naming the reply keyboard this branch deleted).
func BackToTasks(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📋 Задачи", "📋 Tasks"), "top_tasks"),
		),
	)
}

func SettingsMenu(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🕘 Рабочие часы", "🕘 Work hours"), "settings_workhours"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔤 Ключевые слова", "🔤 Keywords"), "settings_keywords"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🌐 Язык", "🌐 Language"), "settings_lang"),
		),
		tgbotapi.NewInlineKeyboardRow(
			// Onboarding again, on demand — language, clock, and how to write.
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❓ Как пользоваться", "❓ How to use"), "ob_start"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu"),
		),
	)
}

// WorkHoursStart / WorkHoursEnd lay the offered hours out three to a row.
//
// The hour list is a parameter rather than a constant here so the handler owns
// what is offered and the keyboard owns only how it looks — and so a test can
// hand it an absurd list and watch the row-packing hold.
func WorkHoursStart(lang i18n.Lang, hours []int) tgbotapi.InlineKeyboardMarkup {
	return hourKeyboard(lang, hours, "wh_start_", "settings_menu")
}

func WorkHoursEnd(lang i18n.Lang, hours []int) tgbotapi.InlineKeyboardMarkup {
	return hourKeyboard(lang, hours, "wh_end_", "settings_menu")
}

func hourKeyboard(lang i18n.Lang, hours []int, prefix, back string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, h := range hours {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%02d:00", h), prefix+strconv.Itoa(h)))
		if len(row) == 3 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), back),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// MonthGrid renders the six-week month.
//
// It receives labels and dates already computed rather than a year and a month
// it would have to walk itself: the marks (today, has-events) depend on data
// this package has no business fetching, and splitting it this way lets the
// layout be tested against a fixed grid instead of against the current date.
//
// weekdayNoop is the header row. Telegram has no inert button, so the cells
// carry "noop" — which the router must accept and ignore. A callback nothing
// answers leaves the spinner turning on the user's screen.
//
// ⚠ monthName arrives already translated: the handler knows the language and
// the month, and building the name here would mean a second calendar.
func MonthGrid(lang i18n.Lang, year, month int, monthName string, labels, dates []string, todayISO string) tgbotapi.InlineKeyboardMarkup {
	pos := strconv.Itoa(year) + "_" + strconv.Itoa(month)

	weekdays := [7]string{
		i18n.T(lang, "Пн", "Mo"), i18n.T(lang, "Вт", "Tu"), i18n.T(lang, "Ср", "We"),
		i18n.T(lang, "Чт", "Th"), i18n.T(lang, "Пт", "Fr"), i18n.T(lang, "Сб", "Sa"),
		i18n.T(lang, "Вс", "Su"),
	}
	header := make([]tgbotapi.InlineKeyboardButton, 0, 7)
	for _, w := range weekdays {
		header = append(header, tgbotapi.NewInlineKeyboardButtonData(w, "noop"))
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅", "cal_prev_"+pos),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", monthName, year), "noop"),
			tgbotapi.NewInlineKeyboardButtonData("➡", "cal_next_"+pos),
		),
		header,
	}

	for i := 0; i+7 <= len(labels) && i+7 <= len(dates); i += 7 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+7; j++ {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(labels[j], "cal_day_"+dates[j]))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Сегодня", "📅 Today"), "cal_day_"+todayISO),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayActions is the day screen. Paging first, because that is the regression
// people feel: v0.2.1 let you walk day to day from here (handler calendar_day_,
// index.mjs:750) and this bot made you go back to the grid every time.
func DayActions(lang i18n.Lang, date, prev, next string, year, month int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅", "cal_day_"+prev),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня", "Today"), "cal_today"),
			tgbotapi.NewInlineKeyboardButtonData("➡", "cal_day_"+next),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.T(lang, "➕ Событие на этот день", "➕ Event on this day"), "cal_new_"+date),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« К месяцу", "« To month"),
				"cal_back_"+strconv.Itoa(year)+"_"+strconv.Itoa(month)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
		),
	)
}
