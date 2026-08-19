package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Scheduling a task onto the calendar from the phone.
//
// This is the one thing the product's own claim rests on — "the calendar is
// truth; tasks exist to support scheduling" — and it was the largest of the
// three capabilities the Go rewrite dropped from the v0.2.1 bot, where it lived
// as task_schedule_* / schedule_task_*_<minutes> (index.mjs:818, 870).
//
// Two taps, in the order people decide: when, then how long. The reverse order
// was v0.2.1's ("Schedule Now" jumped straight to durations), and it forces a
// commitment to today-right-now before the cheap question has been asked.

// scheduleSlots is the closed set of answers TaskScheduleWhen can produce.
// Callback data arrives from the user's client and is not trustworthy: an
// unknown slot must be refused, not defaulted, or a malformed button writes an
// arbitrary time into a calendar.
var scheduleSlots = map[string]bool{"now": true, "hour": true, "eve": true, "tmr": true}

// scheduleStart resolves a slot against the moment the button was pressed.
// Split out from the handler so it can be tested without a bot, a store, or a
// clock: `now` is a parameter precisely so "this evening, pressed at 21:00" is
// a table row rather than a thing to reason about.
func scheduleStart(slot string, now time.Time, loc *time.Location) (time.Time, bool) {
	now = now.In(loc)
	switch slot {
	case "now":
		return now.Truncate(time.Minute), true
	case "hour":
		return now.Add(time.Hour).Truncate(time.Minute), true
	case "eve":
		start := time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, loc)
		// Asking for "this evening" after 19:00 means the next one — the same
		// rule the event flow uses, deliberately: two paths that resolve the
		// same words differently is a bug report waiting to be written.
		if !start.After(now) {
			start = start.AddDate(0, 0, 1)
		}
		return start, true
	case "tmr":
		return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1), true
	default:
		return time.Time{}, false
	}
}

// parsePlanCallback splits task_plan_<uuid>_<slot>_<minutes>.
//
// It cuts from the RIGHT. A task id is a UUID today, but the id is the only
// field here that is opaque to this bot, and splitting left-to-right would make
// every future id format that contains an underscore silently address the wrong
// task. Slot and minutes are ours and known-shaped; the id is whatever is left.
func parsePlanCallback(data string) (taskID, slot string, minutes int, ok bool) {
	cut := strings.LastIndex(data, "_")
	if cut < 0 {
		return "", "", 0, false
	}
	minStr := data[cut+1:]
	data = data[:cut]

	cut = strings.LastIndex(data, "_")
	if cut < 0 {
		return "", "", 0, false
	}
	slot, taskID = data[cut+1:], data[:cut]

	minutes, err := strconv.Atoi(minStr)
	if err != nil || minutes <= 0 || minutes > 24*60 {
		return "", "", 0, false
	}
	if !scheduleSlots[slot] {
		return "", "", 0, false
	}
	if taskID == "" {
		return "", "", 0, false
	}
	return taskID, slot, minutes, true
}

// handleTaskScheduleWhen answers ⏰ Запланировать on the task card.
func (h *Handler) handleTaskScheduleWhen(chatID int64, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	h.sendHTMLWithKeyboard(chatID,
		fmt.Sprintf("⏰ <b>%s</b>\n\nКогда заняться?", format.Escape(title)),
		keyboards.TaskScheduleWhen(taskID))
}

// handleTaskScheduleDuration asks how long, once the slot is chosen.
func (h *Handler) handleTaskScheduleDuration(chatID int64, data string) {
	idx := strings.LastIndex(data, "_")
	if idx < 0 {
		return
	}
	taskID, slot := data[:idx], data[idx+1:]
	if !scheduleSlots[slot] || taskID == "" {
		return
	}

	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}

	loc := h.location()
	start, resolved := scheduleStart(slot, time.Now(), loc)
	if !resolved {
		return
	}

	h.sendHTMLWithKeyboard(chatID,
		fmt.Sprintf("⏰ <b>%s</b>\n🕐 %s\n\nНасколько?",
			format.Escape(title), humanDay(start, time.Now().In(loc))),
		keyboards.TaskScheduleDuration(taskID, slot))
}

// handleTaskSchedule performs the scheduling.
func (h *Handler) handleTaskSchedule(chatID int64, data string) {
	taskID, slot, minutes, ok := parsePlanCallback(data)
	if !ok {
		h.sendHTMLWithKeyboard(chatID, "Не понял кнопку.", keyboards.BackToTasks())
		return
	}

	loc := h.location()
	start, resolved := scheduleStart(slot, time.Now(), loc)
	if !resolved {
		return
	}
	end := start.Add(time.Duration(minutes) * time.Minute)

	us := h.store.GetOrCreate(chatID)
	err := h.api.ScheduleTask(us.AuthToken, taskID,
		start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		h.sendText(chatID, "❌ Не удалось запланировать: "+err.Error())
		return
	}

	h.sendHTML(chatID, fmt.Sprintf("✅ <b>В календаре</b>\n🕐 %s",
		humanRange(start, end, time.Now().In(loc))))
}

// taskTitle looks a task up by id so the keyboards can name what they are about.
//
// It reports failure to the user rather than returning a blank: a card that
// says "Когда заняться?" over an empty title is how someone schedules the wrong
// task. The status filter is empty on purpose — a task already SCHEDULED can be
// moved, and filtering to TODO would make its card vanish mid-flow.
func (h *Handler) taskTitle(chatID int64, taskID string) (string, bool) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.sendText(chatID, "❌ Не удалось загрузить задачу")
		return "", false
	}
	for _, t := range tasks {
		if t.ID == taskID {
			return t.Title, true
		}
	}
	h.sendText(chatID, "Задача не найдена — возможно, она уже удалена.")
	return "", false
}

// humanDay names the day the way the confirmation will, so the question and the
// answer describe the same moment in the same words.
func humanDay(t, now time.Time) string {
	day := func(x time.Time) time.Time {
		return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location())
	}
	hhmm := t.Format("15:04")
	switch days := int(day(t).Sub(day(now)).Hours() / 24); {
	case days == 0:
		return "Сегодня " + hhmm
	case days == 1:
		return "Завтра " + hhmm
	default:
		return fmt.Sprintf("%d %s %s", t.Day(), monthGenitive(t.Month()), hhmm)
	}
}
