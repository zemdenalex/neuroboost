package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// «Задачи дня» in the bot (spec 2026-09-22 §8, plan 2026-09-23 D2). The screen
// asks the API for the day and draws what it says: the level and the count are
// the server's (spec §3), never recomputed here.

const dayTaskDateFlow = "day_task_date"

// dayTitleLimit keeps a task title from swallowing its button.
const dayTitleLimit = 32

// userToday is local midnight of the user's today, in the user's zone.
func (h *Handler) userToday(chatID int64) time.Time {
	now := time.Now().In(h.location(chatID))
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func (h *Handler) parseDay(chatID int64, s string) (time.Time, bool) {
	if s == "today" {
		return h.userToday(chatID), true
	}
	d, err := time.ParseInLocation("2006-01-02", s, h.location(chatID))
	return d, err == nil
}

// splitDayID splits «2026-09-23_<uuid>».
func splitDayID(rest string) (string, string, bool) {
	if len(rest) < 12 || rest[10] != '_' {
		return "", "", false
	}
	return rest[:10], rest[11:], true
}

// dayTasksErrorText is this screen's own reading of the API's codes, asked
// before the general errorText. 🔴 NOT_AN_OCCURRENCE means «the series skips
// that day» here, while errorText reads the same code as «the series ended».
func (h *Handler) dayTasksErrorText(chatID int64, err error) string {
	switch api.CodeOf(err) {
	case "TOO_LATE":
		return h.t(chatID,
			"Убрать из сегодняшнего дня можно только до 12:00, а прошедший день уже не меняется.",
			"A task can leave today only until 12:00, and a past day no longer changes.")
	case "NOT_OPEN":
		return h.t(chatID,
			"Эта задача уже закрыта, её нельзя взять на день.",
			"This task is already closed; it cannot be taken for a day.")
	case "NOT_AN_OCCURRENCE":
		return h.t(chatID,
			"В этот день эта серия не повторяется.",
			"This series does not repeat on that day.")
	case "TASK_NOT_FOUND":
		return h.t(chatID,
			"Этой задачи больше нет, или она тебе больше не видна.",
			"This task no longer exists, or you can no longer see it.")
	}
	return h.errorText(chatID, err)
}

// handleDayTasksCallback answers every dt_ button. Returns false for anything
// else.
func (h *Handler) handleDayTasksCallback(chatID int64, messageID int, data string) bool {
	if !strings.HasPrefix(data, "dt_") {
		return false
	}
	us := h.store.GetOrCreate(chatID)
	withDay := func(prefix string, then func(time.Time)) {
		if day, ok := h.parseDay(chatID, strings.TrimPrefix(data, prefix)); ok {
			then(day)
		}
	}
	withDayID := func(prefix string, then func(time.Time, string)) {
		d, id, ok := splitDayID(strings.TrimPrefix(data, prefix))
		if !ok {
			return
		}
		if day, ok := h.parseDay(chatID, d); ok {
			then(day, id)
		}
	}

	switch {
	case strings.HasPrefix(data, "dt_d_"):
		withDay("dt_d_", func(day time.Time) { h.showDay(chatID, messageID, day) })
	case strings.HasPrefix(data, "dt_ok_"):
		h.tickDayTask(chatID, messageID, strings.TrimPrefix(data, "dt_ok_"))
	case strings.HasPrefix(data, "dt_takeset_"):
		withDay("dt_takeset_", func(day time.Time) { h.takeDay(chatID, messageID, day, false) })
	case strings.HasPrefix(data, "dt_take_"):
		withDay("dt_take_", func(day time.Time) { h.takeDay(chatID, messageID, day, true) })
	case strings.HasPrefix(data, "dt_edit_"):
		withDay("dt_edit_", func(day time.Time) { h.showDayEdit(chatID, messageID, day, "") })
	case strings.HasPrefix(data, "dt_rm_"):
		withDayID("dt_rm_", func(day time.Time, id string) {
			note := ""
			if err := h.api.RemoveDayTask(us.AuthToken, day.Format("2006-01-02"), id); err != nil {
				note = "❌ " + h.dayTasksErrorText(chatID, err)
			}
			h.showDayEdit(chatID, messageID, day, note)
		})
	case strings.HasPrefix(data, "dt_add_"):
		withDay("dt_add_", func(day time.Time) { h.showDayAddList(chatID, messageID, day) })
	case strings.HasPrefix(data, "dt_put_"):
		withDayID("dt_put_", func(day time.Time, id string) {
			note := ""
			if _, err := h.api.AddDayTask(us.AuthToken, day.Format("2006-01-02"), id); err != nil {
				note = "❌ " + h.dayTasksErrorText(chatID, err)
			}
			h.showDayEdit(chatID, messageID, day, note)
		})
	case strings.HasPrefix(data, "dt_pin_"):
		h.showDayPin(chatID, messageID, strings.TrimPrefix(data, "dt_pin_"))
	case strings.HasPrefix(data, "dt_pdt_"):
		// ✏️ Дата — the answer is a typed line (Denis, 23.09).
		us.CurrentFlow, us.FlowStep = dayTaskDateFlow, "date"
		us.FlowData = map[string]any{"task": strings.TrimPrefix(data, "dt_pdt_")}
		h.editOrSend(chatID, messageID, h.t(chatID,
			"📌 На какой день? Напиши дату, например «пятница» или «25.09».",
			"📌 Which day? Write a date, «friday» or «25.09» for instance."), keyboards.None())
	case strings.HasPrefix(data, "dt_pd_"):
		withDayID("dt_pd_", func(day time.Time, id string) { h.pinTask(chatID, messageID, day, id) })
	}
	return true
}

// showDay draws one day. A day not yet taken shows the offer instead of the set.
func (h *Handler) showDay(chatID int64, messageID int, day time.Time) {
	us := h.store.GetOrCreate(chatID)
	iso := day.Format("2006-01-02")
	days, err := h.api.DayTasks(us.AuthToken, iso, iso)
	if err != nil || len(days) == 0 {
		h.editOrSend(chatID, messageID, "❌ "+h.dayTasksErrorText(chatID, err), keyboards.HomeInline(h.lang(chatID)))
		return
	}
	d := days[0]
	today := h.userToday(chatID)
	past := day.Before(today)

	var proposal []api.DayItem
	if !d.Confirmed && !past {
		proposal, _ = h.api.DayProposal(us.AuthToken, iso)
	}
	view := keyboards.DayView{
		Day:     iso,
		Prev:    day.AddDate(0, 0, -1).Format("2006-01-02"),
		Next:    day.AddDate(0, 0, 1).Format("2006-01-02"),
		Items:   dayButtons(d.Items),
		Taken:   d.Confirmed,
		Today:   day.Equal(today),
		Past:    past,
		CanTake: len(proposal) > 0,
	}
	h.editOrSend(chatID, messageID, renderDayScreen(h.lang(chatID), d, proposal, day, today),
		keyboards.DayScreen(h.lang(chatID), view))
}

func dayButtons(items []api.DayItem) []keyboards.DayButton {
	out := make([]keyboards.DayButton, 0, len(items))
	for _, it := range items {
		out = append(out, keyboards.DayButton{ID: it.TaskID, Title: shorten(it.Title, dayTitleLimit), Done: it.Done})
	}
	return out
}

// tickDayTask is ⬜ → ✅ on today's screen (Denis, 23.09). A series closes its
// day through the series door; a one-off closes the task.
func (h *Handler) tickDayTask(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	var err error
	if h.taskRepeats(chatID, taskID) {
		_, err = h.api.MarkOccurrence(us.AuthToken, taskID, "done", 0)
	} else {
		err = h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"status": "DONE"})
	}
	if err != nil {
		h.sendText(chatID, "❌ "+h.dayTasksErrorText(chatID, err))
	}
	h.showDay(chatID, messageID, h.userToday(chatID))
}

// takeDay confirms the day: with the offer as it is at the press (offer), or
// with the set built by hand.
func (h *Handler) takeDay(chatID int64, messageID int, day time.Time, offer bool) {
	us := h.store.GetOrCreate(chatID)
	iso := day.Format("2006-01-02")
	var ids []string
	if offer {
		items, err := h.api.DayProposal(us.AuthToken, iso)
		if err != nil {
			h.sendText(chatID, "❌ "+h.dayTasksErrorText(chatID, err))
			return
		}
		for _, it := range items {
			ids = append(ids, it.TaskID)
		}
	} else {
		days, err := h.api.DayTasks(us.AuthToken, iso, iso)
		if err != nil || len(days) == 0 {
			h.sendText(chatID, "❌ "+h.dayTasksErrorText(chatID, err))
			return
		}
		for _, it := range days[0].Items {
			ids = append(ids, it.TaskID)
		}
	}
	if _, err := h.api.ConfirmDay(us.AuthToken, iso, ids); err != nil {
		h.sendText(chatID, "❌ "+h.dayTasksErrorText(chatID, err))
	}
	h.showDay(chatID, messageID, day)
}

// showDayEdit is «Поменять». note, when set, says why the last press did not
// work — above the screen, not instead of it.
func (h *Handler) showDayEdit(chatID int64, messageID int, day time.Time, note string) {
	us := h.store.GetOrCreate(chatID)
	iso := day.Format("2006-01-02")
	days, err := h.api.DayTasks(us.AuthToken, iso, iso)
	if err != nil || len(days) == 0 {
		h.editOrSend(chatID, messageID, "❌ "+h.dayTasksErrorText(chatID, err), keyboards.HomeInline(h.lang(chatID)))
		return
	}
	text := dayTitle(h.lang(chatID), day) + "\n\n" +
		h.t(chatID, "❌ убирает задачу из дня, ➕ добавляет.", "❌ takes a task out of the day, ➕ adds one.")
	if note != "" {
		text = note + "\n\n" + text
	}
	h.editOrSend(chatID, messageID, text,
		keyboards.DayEdit(h.lang(chatID), iso, dayButtons(days[0].Items), days[0].Confirmed))
}

// dayAddLimit is how many open tasks ➕ offers at once.
const dayAddLimit = 8

// showDayAddList offers open tasks not already in the day, most urgent first
// (1 is the most urgent, 0 the buffer, last — gotcha 4).
func (h *Handler) showDayAddList(chatID int64, messageID int, day time.Time) {
	us := h.store.GetOrCreate(chatID)
	iso := day.Format("2006-01-02")
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.editOrSend(chatID, messageID, "❌ "+h.dayTasksErrorText(chatID, err), keyboards.HomeInline(h.lang(chatID)))
		return
	}
	in := map[string]bool{}
	if days, err := h.api.DayTasks(us.AuthToken, iso, iso); err == nil && len(days) > 0 {
		for _, it := range days[0].Items {
			in[it.TaskID] = true
		}
	}
	var open []api.Task
	for _, t := range tasks {
		if in[t.ID] || t.Status == "DONE" || t.Status == "CANCELLED" {
			continue
		}
		open = append(open, t)
	}
	rank := func(p int) int {
		if p == 0 {
			return 99
		}
		return p
	}
	sort.SliceStable(open, func(i, j int) bool { return rank(open[i].Priority) < rank(open[j].Priority) })
	if len(open) > dayAddLimit {
		open = open[:dayAddLimit]
	}
	var buttons []keyboards.DayButton
	for _, t := range open {
		buttons = append(buttons, keyboards.DayButton{ID: t.ID, Title: shorten(t.Title, dayTitleLimit)})
	}
	text := h.t(chatID, "➕ Что добавить?", "➕ What should I add?")
	if len(buttons) == 0 {
		text = h.t(chatID, "Открытых задач, которых ещё нет в этом дне, нет.", "There are no open tasks that are not in this day yet.")
	}
	h.editOrSend(chatID, messageID, dayTitle(h.lang(chatID), day)+"\n\n"+text,
		keyboards.DayAddList(h.lang(chatID), iso, buttons))
}

// showDayPin is «📌 В задачи дня» on a task card.
func (h *Handler) showDayPin(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	today := h.userToday(chatID)
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "📌 <b>%s</b>\n\nНа какой день взять?", "📌 <b>%s</b>\n\nWhich day should it go to?"), format.Escape(title)),
		keyboards.DayPinPick(h.lang(chatID), taskID, today.Format("2006-01-02"), today.AddDate(0, 0, 1).Format("2006-01-02")))
}

// pinTask adds the task to a day and says where it went.
func (h *Handler) pinTask(chatID int64, messageID int, day time.Time, taskID string) {
	us := h.store.GetOrCreate(chatID)
	iso := day.Format("2006-01-02")
	if _, err := h.api.AddDayTask(us.AuthToken, iso, taskID); err != nil {
		h.editOrSend(chatID, messageID, "❌ "+h.dayTasksErrorText(chatID, err), keyboards.DayPinned(h.lang(chatID), taskID, iso))
		return
	}
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "📌 Добавлено в задачи дня: %s", "📌 Added to the day tasks: %s"), dayLabel(h.lang(chatID), day)),
		keyboards.DayPinned(h.lang(chatID), taskID, iso))
}

// handleDayTaskDate is the typed answer to «✏️ Дата». A past or unreadable date
// asks again and writes nothing.
func (h *Handler) handleDayTaskDate(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	taskID, _ := us.FlowData["task"].(string)
	now := time.Now().In(h.location(chatID))
	d := parse.ParseLine(text, now).Draft
	if !d.HasDay {
		h.sendText(chatID, h.t(chatID,
			"Не понял дату. Напиши, например, «пятница» или «25.09».",
			"I did not get the date. Write «friday» or «25.09», for instance."))
		return
	}
	day := time.Date(d.Day.Year(), d.Day.Month(), d.Day.Day(), 0, 0, 0, 0, h.location(chatID))
	if day.Before(h.userToday(chatID)) {
		h.sendText(chatID, h.t(chatID,
			"Это прошлый день, его уже не поменять. Напиши сегодняшнюю или будущую дату.",
			"That day is past and no longer changes. Write today or a later date."))
		return
	}
	h.store.ClearFlow(chatID)
	h.pinTask(chatID, 0, day, taskID)
}
