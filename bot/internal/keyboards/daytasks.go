package keyboards

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// «Задачи дня» (spec 2026-09-22 §8, plan 2026-09-23). Every callback is dt_…
// and routed by handlers.handleDayTasksCallback; dtn_ is the target setting.
// Titles arrive shortened by the handler, which owns shorten().

// DayButton is one task on a day screen.
type DayButton struct {
	ID    string
	Title string
	Done  bool
}

// DayView is what the day screen's buttons depend on. Days are YYYY-MM-DD.
type DayView struct {
	Day, Prev, Next string
	Items           []DayButton
	Taken           bool // the day was confirmed
	Today           bool // the day is the user's today — only then ⬜ ticks
	Past            bool // read only
	CanTake         bool // an untaken day has a non-empty offer
}

// DayScreen is 📌 Задачи дня. Denis 23.09: a press on ⬜ today marks it done;
// an untaken day offers «✅ Беру» and «✏️ Выбрать самому».
func DayScreen(lang i18n.Lang, v DayView) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if v.Taken {
		for _, it := range v.Items {
			label, data := "⬜ "+it.Title, "task_action_"+it.ID
			if it.Done {
				label = "✅ " + it.Title
			} else if v.Today {
				// The day travels with the press (review I1): an old message
				// must not close a later day.
				data = "dt_ok_" + v.Day + "_" + it.ID
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(label, data)))
		}
	}
	if !v.Past {
		switch {
		case v.Taken:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Поменять", "✏️ Change"), "dt_edit_"+v.Day)))
		case v.CanTake:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Беру", "✅ Take it"), "dt_take_"+v.Day),
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Выбрать самому", "✏️ Choose myself"), "dt_edit_"+v.Day)))
		default:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Выбрать самому", "✏️ Choose myself"), "dt_edit_"+v.Day)))
		}
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀", "dt_d_"+v.Prev),
			tgbotapi.NewInlineKeyboardButtonData("▶", "dt_d_"+v.Next)),
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu")))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayEdit is «Поменять»: ❌ takes a task out, ➕ adds, and an untaken day with
// something in it can be taken as it stands.
func DayEdit(lang i18n.Lang, day string, items []DayButton, taken bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, it := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ "+it.Title, "dt_rm_"+day+"_"+it.ID)))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Добавить", "➕ Add"), "dt_add_"+day)))
	if !taken && len(items) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Беру этот набор", "✅ Take this set"), "dt_takeset_"+day)))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, HelpDayTasks),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "dt_d_"+day)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayAddList offers open tasks not yet in the day.
func DayAddList(lang i18n.Lang, day string, items []DayButton) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, it := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ "+it.Title, "dt_put_"+day+"_"+it.ID)))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, HelpDayTasks),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "dt_edit_"+day)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayPinPick is «📌 В задачи дня» on a task card. Denis 23.09: today,
// tomorrow, or a typed date.
func DayPinPick(lang i18n.Lang, taskID, today, tomorrow string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Сегодня", "Today"), "dt_pd_"+today+"_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Завтра", "Tomorrow"), "dt_pd_"+tomorrow+"_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Дата", "✏️ Date"), "dt_pdt_"+taskID)),
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« К задаче", "« To the task"), "task_action_"+taskID)))
}

// DayPinned answers a pin: to that day's screen, or back to the task.
func DayPinned(lang i18n.Lang, taskID, day string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📌 Задачи дня", "📌 Day tasks"), "dt_d_"+day),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« К задаче", "« To the task"), "task_action_"+taskID)),
		tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpDayTasks)))
}

// DayDateCancel sits under «✏️ Дата»: the way out of the typed-date step.
func DayDateCancel(lang i18n.Lang, taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, HelpDayTasks),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "dt_pin_"+taskID)))
}

// DaySettingsView is what ⚙️ → 📌 Задачи дня shows (spec 2026-09-22 §11).
type DaySettingsView struct {
	On, PaintBefore bool
	Target          int
	// Cell is what a month cell shows: "both" (or ""), "colour", "bar".
	Cell string
}

// cellRow is the choice of what a month cell shows (spec §6), current ticked.
func cellRow(current string) []tgbotapi.InlineKeyboardButton {
	if current == "" {
		current = "both"
	}
	var row []tgbotapi.InlineKeyboardButton
	for _, c := range []struct{ key, label string }{{"both", "🟩 ▅"}, {"colour", "🟩"}, {"bar", "▅"}} {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(tick(c.key == current)+c.label, "dts_cell_"+c.key))
	}
	return row
}

// targetRow is 3…7 with the current one ticked, under a callback prefix.
func targetRow(current int, prefix string) []tgbotapi.InlineKeyboardButton {
	var row []tgbotapi.InlineKeyboardButton
	for n := 3; n <= 7; n++ {
		s := strconv.Itoa(n)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(tick(n == current)+s, prefix+s))
	}
	return row
}

// DaySettings is ⚙️ → 📌 Задачи дня: the switch, and when on, the target and
// whether days before the start are coloured. Off leaves only the way back on.
func DaySettings(lang i18n.Lang, v DaySettingsView) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if !v.On {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Выключены · включить", "❌ Off · switch on"), "dts_on")))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Включены", "✅ On"), "dts_off")))
		rows = append(rows, targetRow(v.Target, "dtn_"))
		paint := tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🎨 Дни до начала: не красить", "🎨 Days before the start: no colour"), "dts_pb_on")
		if v.PaintBefore {
			paint = tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🎨 Дни до начала: ⬛", "🎨 Days before the start: ⬛"), "dts_pb_off")
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(paint))
		rows = append(rows, cellRow(v.Cell))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, HelpDayTasks),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Настройки", "« Settings"), "settings_menu")))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// DayTargetPick is the 3…7 row alone, with its own prefix and a «next» button:
// onboarding asks N right after «✅ Включить» (spec §11).
func DayTargetPick(lang i18n.Lang, current int, prefix, nextData, nextLabel string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(targetRow(current, prefix),
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(nextLabel, nextData)))
}

// DayTasksOff answers an old 📌 button while day tasks are switched off: the
// way back on, and the menu (spec §11).
func DayTasksOff(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Включить", "✅ Switch on"), "dts_on")),
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu")))
}

// DayOnboard is the day-tasks question: in onboarding (ob_dt_on / ob_dt_off)
// and once for people onboarded earlier (dtq_on / dtq_off), spec §11.
func DayOnboard(lang i18n.Lang, onData, offData string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Включить", "✅ Switch on"), onData),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Не сейчас", "Not now"), offData)),
		tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpDayTasks)))
}

// DaySettingsRetry answers ⚙️ → 📌 when the settings could not be read: try
// again, or back to settings (review M2).
func DaySettingsRetry(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔄 Ещё раз", "🔄 Try again"), "settings_dtn")),
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Настройки", "« Settings"), "settings_menu")))
}
