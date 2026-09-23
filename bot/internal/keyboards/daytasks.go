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

// DayTarget is ⚙️ → 🎯 Задач в день: 3…7, the current one ticked.
func DayTarget(lang i18n.Lang, current int) tgbotapi.InlineKeyboardMarkup {
	var row []tgbotapi.InlineKeyboardButton
	for n := 3; n <= 7; n++ {
		s := strconv.Itoa(n)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(tick(n == current)+s, "dtn_"+s))
	}
	return tgbotapi.NewInlineKeyboardMarkup(row,
		tgbotapi.NewInlineKeyboardRow(
			HelpButton(lang, HelpDayTasks),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Настройки", "« Settings"), "settings_menu")))
}
