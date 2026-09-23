package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// Quick save — Denis, 23.09: a line typed without a command becomes a task
// at once, «купить молоко завтра» → saved, in one message, no press.
//
// Before this the same line took two presses: «Что создать?» → 📋 Задача →
// ✅ Создать. The kind question is not gone, it moved: 📅 → Событие and
// 📝 → Заметка under the saved task undo the guess when it was wrong, which
// for a line with no clock time it rarely is.

// plainTaskLine is a line that needs no question: no clock time (task or
// event is a real choice there — a timed task is bound to an event), not a
// list (one or many is a real choice), and no «повтор» without a frequency
// (the card promises to ask). Returns what the task parser made of it.
func (h *Handler) plainTaskLine(chatID int64, text string) (parse.TaskResult, bool) {
	now := time.Now().In(h.location(chatID))
	if parse.ParseLine(text, now).Draft.HasTime || parse.LooksLikeList(text, now) {
		return parse.TaskResult{}, false
	}
	r := parse.ParseTask(text, now)
	if r.RepeatAsked || strings.TrimSpace(r.Title) == "" {
		return parse.TaskResult{}, false
	}
	return r, true
}

// taskReqFrom is everything the task vocabulary understood, not just the
// title — a word the user typed and the bot dropped is a word ignored.
func taskReqFrom(r parse.TaskResult) api.CreateTaskReq {
	req := api.CreateTaskReq{Title: r.Title, Status: "TODO", Priority: r.Priority,
		EstimatedMinutes: r.EstimatedMinutes}
	if r.DueDate != nil {
		due := r.DueDate.Format(time.RFC3339)
		req.DueDate = &due
	}
	if len(r.Tags) > 0 {
		req.Tags = r.Tags
	}
	if r.Rrule != "" {
		rule := r.Rrule
		req.Rrule = &rule
	}
	return req
}

func (h *Handler) quickSaveTask(chatID int64, raw string, r parse.TaskResult) {
	us := h.store.GetOrCreate(chatID)
	task, err := h.api.CreateTask(us.AuthToken, taskReqFrom(r))
	h.store.ClearFlow(chatID)
	if err != nil {
		h.sendHTMLWithKeyboard(chatID,
			h.t(chatID, "❌ Не удалось создать: ", "❌ Could not create: ")+h.errorText(chatID, err),
			keyboards.HomeInline(h.lang(chatID)))
		return
	}
	us.QuickTaskID, us.QuickRaw = task.ID, raw

	// The card the old path showed before saving, now as the receipt: the
	// user sees what was understood (due, priority, repeat) and can fix it.
	text := h.t(chatID, "✅ <b>Задача создана</b>\n", "✅ <b>Task created</b>\n") +
		taskCardText(h.lang(chatID), r, h.timezone(chatID), h.priorityStyle(chatID))
	h.sendHTMLWithKeyboard(chatID, text, keyboards.QuickSaved(h.lang(chatID), task.ID))
}

// handleQuickSavedCallback answers the qs_ buttons under a saved task.
// Returns false for anything else.
func (h *Handler) handleQuickSavedCallback(chatID int64, messageID int, data string) bool {
	var action, taskID string
	for _, a := range []string{"undo", "event", "note"} {
		if p := "qs_" + a + "_"; strings.HasPrefix(data, p) {
			action, taskID = a, strings.TrimPrefix(data, p)
		}
	}
	if action == "" {
		return false
	}
	us := h.store.GetOrCreate(chatID)

	// The line the task came from. Only the LAST quick task is remembered;
	// for an older one the saved title is what there is.
	raw := us.QuickRaw
	if us.QuickTaskID != taskID {
		title, ok := h.taskTitle(chatID, taskID)
		if !ok {
			return true
		}
		raw = title
	}

	if err := h.api.DeleteTask(us.AuthToken, taskID); err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не получилось: ", "❌ That didn't work: ")+h.errorText(chatID, err),
			keyboards.HomeInline(h.lang(chatID)))
		return true
	}
	if us.QuickTaskID == taskID {
		us.QuickTaskID, us.QuickRaw = "", ""
	}

	switch action {
	case "undo":
		h.editOrSend(chatID, messageID,
			fmt.Sprintf(h.t(chatID, "↩️ Отменено: %s", "↩️ Undone: %s"), format.Escape(raw)),
			keyboards.HomeInline(h.lang(chatID)))
	case "event":
		// Chosen as an event: the same door as «📅 Событие» on the old question.
		h.quickKind = kindEvent
		h.store.GetOrCreate(chatID).FlowData = map[string]any{"raw": raw}
		h.quickAs(chatID, "event", raw)
	case "note":
		h.store.GetOrCreate(chatID).FlowData = map[string]any{"raw": raw}
		h.quickAs(chatID, "note", raw)
	}
	return true
}
