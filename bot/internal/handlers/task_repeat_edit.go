package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Turning a repeat on, off, or into something else — on a task that already
// exists.
//
// 🔴 The API has taken `rrule` on PATCH since 20.09, and the bot had no
// button, so a repeat could be created and never undone. Denis wrote «Надо»
// against exactly that line on 21.09.
//
// ⚠ Same class as the four possibilities of 19.08 that were «written, tested,
// and never reachable»: the endpoint existing is not the feature existing.
// Кнопка — заявка, обработчик — свидетельство.

// taskByID reads one task from the list the API will give us.
//
// Asked of the server rather than remembered from the card: a card in the chat
// may be hours old, and the value it was drawn with may have been changed from
// the web since. A miss returns false rather than a zero Task, so callers
// cannot mistake «not found» for «no repeat».
func (h *Handler) taskByID(chatID int64, taskID string) (api.Task, bool) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		return api.Task{}, false
	}
	for _, t := range tasks {
		if t.ID == taskID {
			return t, true
		}
	}
	return api.Task{}, false
}

// handleTaskRepeatMenu is task_rp_<id>.
func (h *Handler) handleTaskRepeatMenu(chatID int64, messageID int, taskID string) {
	t, ok := h.taskByID(chatID, taskID)
	if !ok {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Задача не найдена.", "Task not found."),
			keyboards.BackToTasks(h.lang(chatID)))
		return
	}

	text := h.t(chatID, "🔁 <b>Повтор</b>\n\nКак часто возвращать?", "🔁 <b>Repeat</b>\n\nHow often should it come back?")
	if t.Repeats() {
		text = fmt.Sprintf(h.t(chatID,
			"🔁 <b>Повтор</b>\n\nСейчас: %s\n\nКак часто возвращать?",
			"🔁 <b>Repeat</b>\n\nNow: %s\n\nHow often should it come back?"),
			freqName(h.lang(chatID), t.Rrule))
	}
	h.editOrSend(chatID, messageID, text, keyboards.TaskRepeat(h.lang(chatID), taskID, t.Rrule))
}

// handleTaskRepeatSet is task_rpd_<id>_<code>.
func (h *Handler) handleTaskRepeatSet(chatID int64, messageID int, taskID, code string) {
	rule, known := keyboards.RepeatCodes[code]
	if !known {
		return
	}
	us := h.store.GetOrCreate(chatID)

	// 🔴 An empty string, not an absent key. api-go reads `"rrule": ""` as
	// «clear it» and drops repeat_anchor with it; leaving the key out means
	// «don't touch», which is how a «Не повторять» button would silently do
	// nothing while reporting success.
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"rrule": rule}); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}

	if rule == "" {
		h.sendText(chatID, h.t(chatID,
			"🔁 Повтор выключен. Задача осталась обычной, с теми же сроком и оценкой.",
			"🔁 Repeat is off. The task stays, with the same due date and estimate."))
	} else {
		h.sendText(chatID, fmt.Sprintf(h.t(chatID, "🔁 Теперь: %s", "🔁 Now: %s"),
			freqName(h.lang(chatID), rule)))
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

// handleTaskNagMenu is task_ng_<id>.
func (h *Handler) handleTaskNagMenu(chatID int64, messageID int, taskID string) {
	t, ok := h.taskByID(chatID, taskID)
	if !ok {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Задача не найдена.", "Task not found."),
			keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	h.editOrSend(chatID, messageID, h.t(chatID,
		"🔔 <b>Долбить</b>\n\nЕсли напоминание осталось без ответа, через сколько повторить?",
		"🔔 <b>Nag</b>\n\nIf a reminder goes unanswered, how soon should it come back?"),
		keyboards.TaskNag(h.lang(chatID), taskID, t.NagMinutes))
}

// handleTaskNagSet is task_ngd_<id>_<code>.
func (h *Handler) handleTaskNagSet(chatID int64, messageID int, taskID, code string) {
	minutes, known := keyboards.NagCodes[code]
	if !known {
		return
	}
	us := h.store.GetOrCreate(chatID)

	// 0 clears the column — the API reads any non-positive value that way.
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"nag_minutes": minutes}); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}

	if minutes == 0 {
		h.sendText(chatID, h.t(chatID,
			"🔔 Не буду долбить. Напоминание придёт один раз.",
			"🔔 No nagging. The reminder arrives once."))
	} else {
		h.sendText(chatID, fmt.Sprintf(h.t(chatID,
			"🔔 Буду повторять каждые %d мин, пока не ответишь.",
			"🔔 I'll repeat every %d min until you answer."), minutes))
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

// «Свой срок» for postponing a series.
//
// 🔴 The checklist of 20.09 said this was deliberately absent: «она открыла бы
// вопрос, на который этот экран ещё не умеет отвечать». Denis's answer on
// 21.09 was one word — «Нужен» — so the screen learns to answer it.
//
// ⚠ Days, not a date. PostponeSeries takes a count of days from today and
// skips whatever falls inside; a date would have to be turned into that count
// anyway, and «до 5 октября» on a monthly series means something the endpoint
// cannot express. Asking for what the API actually takes keeps the promise
// small enough to keep.
func (h *Handler) handleTaskPostponeCustom(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "postpone_custom"
	us.FlowStep = "text"
	us.FlowData["taskID"] = taskID
	h.editOrSend(chatID, messageID, h.t(chatID,
		"⏰ <b>Отложить</b>\n\nНа сколько дней? Напиши число, например «10».\n\nРитм не сдвинется: эти дни просто будут пропущены.\n\n«cancel» отменяет.",
		"⏰ <b>Postpone</b>\n\nFor how many days? Type a number, «10» for instance.\n\nThe rhythm stays; these days are simply skipped.\n\n«cancel» to stop."),
		keyboards.None())
}

// handlePostponeCustomText answers it.
func (h *Handler) handlePostponeCustomText(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	taskID, _ := us.FlowData["taskID"].(string)

	days, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || days < 1 || days > maxPostponeDays {
		// The draft is NOT thrown away — the same rule the wizard steps follow
		// since 20.09. A number that could not be read is a question still
		// open, not a flow to abandon.
		h.sendText(chatID, fmt.Sprintf(h.t(chatID,
			"Нужно число дней от 1 до %d. Напиши, например, «10» или «cancel».",
			"I need a number of days from 1 to %d. Type «10», say, or «cancel»."), maxPostponeDays))
		return
	}

	h.store.ClearFlow(chatID)
	if taskID == "" {
		h.sendText(chatID, h.t(chatID, "Не помню, какую задачу откладывал.", "I've lost which task that was."))
		return
	}
	h.handleTaskPostponeDays(chatID, 0, taskID, days)
}

// maxPostponeDays is a year. Beyond that the answer is «switch the repeat
// off», not «skip every day until then» — and PostponeSeries walks the days
// one by one, so an unbounded number is also an unbounded loop.
const maxPostponeDays = 365
