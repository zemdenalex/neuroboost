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

// Subtasks in the bot (Denis's plan 23.09, day 2): the card lists them with
// ✅, «➕ Подзадача» adds one, a task with subtasks is a project. The API has
// had parent_id all along; the web draws the tree.

const subtaskFlow = "subtask"

// subtaskView is what the card of taskID shows, from one read of the list.
// It returns the progress line too ("" without subtasks).
func (h *Handler) subtaskView(chatID int64, tasks []api.Task, taskID string) (keyboards.SubtaskView, string) {
	var v keyboards.SubtaskView
	done := 0
	for _, t := range tasks {
		if t.ID == taskID {
			v.ParentID = t.ParentID
		}
		if t.ParentID != taskID {
			continue
		}
		closed := t.Status == "DONE" || (t.Repeats() && t.OccurrenceState == "done")
		if closed {
			done++
		}
		v.Items = append(v.Items, keyboards.SubtaskItem{ID: t.ID, Title: t.Title, Done: closed})
	}
	line := ""
	if len(v.Items) > 0 {
		line = fmt.Sprintf(h.t(chatID, "📁 Подзадачи: %d из %d\n", "📁 Subtasks: %d of %d\n"), done, len(v.Items))
	}
	if v.ParentID != "" {
		for _, t := range tasks {
			if t.ID == v.ParentID {
				line += fmt.Sprintf(h.t(chatID, "↑ Часть задачи: <b>%s</b>\n", "↑ Part of: <b>%s</b>\n"), format.Escape(t.Title))
			}
		}
	}
	return v, line
}

// handleSubtaskAdd asks for the subtask as one line.
func (h *Handler) handleSubtaskAdd(chatID int64, messageID int, parentID string) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось загрузить задачу.", "❌ Could not load the task."))
		return
	}
	var parent *api.Task
	for i := range tasks {
		if tasks[i].ID == parentID {
			parent = &tasks[i]
		}
	}
	if parent == nil {
		h.editOrSend(chatID, messageID, h.t(chatID, "Задача не найдена.", "Task not found."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	us.CurrentFlow, us.FlowStep = subtaskFlow, "title"
	// The calendar travels with the question: the API would otherwise put the
	// subtask in the author's personal calendar.
	us.FlowData = map[string]any{"parent": parentID, "calendar": parent.CalendarID}
	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
		"➕ <b>Подзадача к «%s»</b>\n\nНапиши её одной строкой. Срок и приоритет можно прямо в строке: «купить краску завтра».",
		"➕ <b>Subtask of «%s»</b>\n\nWrite it as one line. A due date and priority can go in the line: «buy paint tomorrow»."),
		format.Escape(parent.Title)), keyboards.SubtaskPrompt(h.lang(chatID), parentID))
}

// handleSubtaskText saves the line as a subtask and shows its task again.
func (h *Handler) handleSubtaskText(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	parentID, _ := us.FlowData["parent"].(string)
	calID, _ := us.FlowData["calendar"].(string)
	r := parse.ParseTask(text, time.Now().In(h.location(chatID)))
	if strings.TrimSpace(r.Title) == "" {
		h.sendText(chatID, h.t(chatID, "Напиши, что сделать, одной строкой.", "Write what to do as one line."))
		return
	}
	req := taskReqFrom(r)
	req.ParentID = &parentID
	if calID != "" {
		req.CalendarID = &calID
	}
	h.store.ClearFlow(chatID)
	if _, err := h.api.CreateTask(us.AuthToken, req); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	h.handleTaskAction(chatID, 0, parentID)
}

// handleSubtaskDone ticks a subtask through the one done path, then redraws
// its task's card in place.
func (h *Handler) handleSubtaskDone(chatID int64, messageID int, subID string) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось загрузить задачу.", "❌ Could not load the task."))
		return
	}
	parentID := ""
	for _, t := range tasks {
		if t.ID == subID {
			parentID = t.ParentID
		}
	}
	if _, err := h.closeTask(chatID, subID); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получилось: ", "❌ That didn't work: ")+h.errorText(chatID, err))
		return
	}
	if parentID == "" {
		h.handleTaskAction(chatID, messageID, subID)
		return
	}
	h.handleTaskAction(chatID, messageID, parentID)
}
