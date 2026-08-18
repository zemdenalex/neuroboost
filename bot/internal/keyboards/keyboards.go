package keyboards

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎯 Today"),
			tgbotapi.NewKeyboardButton("📋 Tasks"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Note"),
			tgbotapi.NewKeyboardButton("➕ New Task"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 New Event"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🗓 Calendar"),
			tgbotapi.NewKeyboardButton("🗂 Planning"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Stats"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Settings"),
		),
	)
}

func TaskPriority() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Emergency", "priority_1"),
			tgbotapi.NewInlineKeyboardButtonData("🟠 Urgent", "priority_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟡 Normal", "priority_3"),
			tgbotapi.NewInlineKeyboardButtonData("🟢 Low", "priority_4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚪ If Possible", "priority_5"),
			tgbotapi.NewInlineKeyboardButtonData("🔵 Buffer", "priority_0"),
		),
	)
}

func TaskActions(taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать", "task_sched_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "task_done_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete", "task_delete_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "top_tasks"),
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
func TaskScheduleWhen(taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сейчас", "task_when_"+taskID+"_now"),
			tgbotapi.NewInlineKeyboardButtonData("Через час", "task_when_"+taskID+"_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня вечером", "task_when_"+taskID+"_eve"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра утром", "task_when_"+taskID+"_tmr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Отмена", "task_action_"+taskID),
		),
	)
}

// TaskScheduleDuration asks how long, carrying the slot forward so the answer
// needs nothing remembered.
func TaskScheduleDuration(taskID, slot string) tgbotapi.InlineKeyboardMarkup {
	at := func(minutes string) tgbotapi.InlineKeyboardButton {
		label := map[string]string{"15": "⏱ 15м", "30": "⏰ 30м", "60": "⏰ 1ч", "120": "⏰ 2ч"}[minutes]
		return tgbotapi.NewInlineKeyboardButtonData(label, "task_plan_"+taskID+"_"+slot+"_"+minutes)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(at("15"), at("30")),
		tgbotapi.NewInlineKeyboardRow(at("60"), at("120")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "task_sched_"+taskID),
		),
	)
}

func BackToMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
		),
	)
}

// EventWhen offers the times people actually pick when they typed a title with
// no time. Deliberately four coarse choices, not a clock: this keyboard exists
// to rescue a half-typed line in one tap, and anything finer is faster to type.
func EventWhen() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сейчас", "when_now"),
			tgbotapi.NewInlineKeyboardButtonData("Через час", "when_hour"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня вечером", "when_evening"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра утром", "when_tomorrow"),
		),
	)
}

// SettingsMenu is the ⚙️ screen. One working control today; the button that
// leads to it must do something, which is the whole point of the screen.
func SettingsMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🕘 Рабочие часы", "settings_workhours"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
		),
	)
}

// WorkHoursStart / WorkHoursEnd lay the offered hours out three to a row.
//
// The hour list is a parameter rather than a constant here so the handler owns
// what is offered and the keyboard owns only how it looks — and so a test can
// hand it an absurd list and watch the row-packing hold.
func WorkHoursStart(hours []int) tgbotapi.InlineKeyboardMarkup {
	return hourKeyboard(hours, "wh_start_", "settings_menu")
}

func WorkHoursEnd(hours []int) tgbotapi.InlineKeyboardMarkup {
	return hourKeyboard(hours, "wh_end_", "settings_menu")
}

func hourKeyboard(hours []int, prefix, back string) tgbotapi.InlineKeyboardMarkup {
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
		tgbotapi.NewInlineKeyboardButtonData("« Назад", back),
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
func MonthGrid(year, month int, monthName string, labels, dates []string) tgbotapi.InlineKeyboardMarkup {
	pos := strconv.Itoa(year) + "_" + strconv.Itoa(month)

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅", "cal_prev_"+pos),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", monthName, year), "noop"),
			tgbotapi.NewInlineKeyboardButtonData("➡", "cal_next_"+pos),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Пн", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Вт", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Ср", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Чт", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Пт", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Сб", "noop"),
			tgbotapi.NewInlineKeyboardButtonData("Вс", "noop"),
		),
	}

	for i := 0; i+7 <= len(labels) && i+7 <= len(dates); i += 7 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+7; j++ {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(labels[j], "cal_day_"+dates[j]))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayView sends the user back to the month they came from, not to a default.
func DayView(year, month int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« К месяцу",
				"cal_back_"+strconv.Itoa(year)+"_"+strconv.Itoa(month)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
		),
	)
}
