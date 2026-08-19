package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func (h *Handler) startNoteFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "note"
	us.FlowStep = "text"
	h.sendText(chatID, "📝 Send me your note:")
}

func (h *Handler) startNewTaskFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_task"
	us.FlowStep = "title"
	h.sendHTML(chatID, "➕ <b>New Task</b>\n\nWhat's the task title?")
}

func (h *Handler) handleFlowInput(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	if strings.ToLower(text) == "cancel" || text == "/start" {
		h.store.ClearFlow(chatID)
		h.handleStart(chatID)
		return
	}

	switch us.CurrentFlow {
	case "note":
		h.handleNoteFlow(chatID, text)
	case "new_task":
		h.handleNewTaskFlow(chatID, text)
	case "new_event":
		h.handleNewEventFlow(chatID, text)
	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.HomeInline())
	}
}

func (h *Handler) handleNoteFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	buffer := 5
	_, err := h.api.CreateTask(us.AuthToken, api.CreateTaskReq{
		Title:    text,
		Priority: &buffer,
		Status:   "TODO",
	})
	h.store.ClearFlow(chatID)
	if err != nil {
		h.sendText(chatID, "❌ Failed to save: "+err.Error())
		return
	}
	h.sendText(chatID, "✅ Note saved as task!")
}

func (h *Handler) handleNewTaskFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	switch us.FlowStep {
	case "title":
		r := parse.ParseTask(text, time.Now().In(h.location()))
		us.FlowData["title"] = r.Title
		if r.Priority != nil {
			us.FlowData["priority"] = *r.Priority
		}
		if r.EstimatedMinutes != nil {
			us.FlowData["minutes"] = *r.EstimatedMinutes
		}
		if r.DueDate != nil {
			us.FlowData["due"] = r.DueDate.Format(time.RFC3339)
		}
		if len(r.Tags) > 0 {
			us.FlowData["tags"] = r.Tags
		}
		us.FlowStep = "card"
		h.sendHTMLWithKeyboard(chatID, taskCardText(r, h.cfg.Timezone), keyboards.TaskCard())
	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.HomeInline())
	}
}

func (h *Handler) handlePrioritySelect(chatID int64, data string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "priority" {
		return
	}

	priorityStr := strings.TrimPrefix(data, "priority_")
	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		return
	}

	title, _ := us.FlowData["title"].(string)
	task, err := h.api.CreateTask(us.AuthToken, api.CreateTaskReq{
		Title:    title,
		Priority: &priority,
		Status:   "TODO",
	})
	h.store.ClearFlow(chatID)

	if err != nil {
		h.sendText(chatID, "❌ Failed to create: "+err.Error())
		return
	}

	h.sendHTML(chatID, fmt.Sprintf("✅ Task created!\n\n<b>%s</b>\nID: <code>%s</code>", title, task.ID))
}

// handleTaskCardSave creates the task from the card's understanding as it
// stands — nothing more is asked, because nothing more is required.
func (h *Handler) handleTaskCardSave(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "card" {
		return
	}

	title, _ := us.FlowData["title"].(string)
	if title == "" {
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Не помню название.", keyboards.HomeInline())
		return
	}

	req := api.CreateTaskReq{Title: title, Status: "TODO"}
	if p, ok := us.FlowData["priority"].(int); ok {
		req.Priority = &p
	}
	if m, ok := us.FlowData["minutes"].(int); ok {
		req.EstimatedMinutes = &m
	}
	if d, ok := us.FlowData["due"].(string); ok && d != "" {
		req.DueDate = &d
	}
	if tags, ok := us.FlowData["tags"].([]string); ok && len(tags) > 0 {
		req.Tags = tags
	}

	task, err := h.api.CreateTask(us.AuthToken, req)
	h.store.ClearFlow(chatID)
	if err != nil {
		h.editOrSend(chatID, messageID, "❌ Не удалось создать: "+err.Error(), keyboards.HomeInline())
		return
	}

	h.editOrSend(chatID, messageID,
		fmt.Sprintf("✅ <b>Задача создана</b>\n%s", format.Escape(task.Title)),
		keyboards.HomeInline())
}
