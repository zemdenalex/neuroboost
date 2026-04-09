package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
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
	default:
		h.store.ClearFlow(chatID)
		h.sendText(chatID, "Something went wrong. Use /start")
	}
}

func (h *Handler) handleNoteFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	_, err := h.api.CreateTask(us.AuthToken, api.CreateTaskReq{
		Title:    text,
		Priority: 5,
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
		us.FlowData["title"] = text
		us.FlowStep = "priority"
		h.sendHTMLWithKeyboard(chatID, "Choose priority:", keyboards.TaskPriority())
	default:
		h.store.ClearFlow(chatID)
		h.sendText(chatID, "Something went wrong. Use /start")
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
		Priority: priority,
		Status:   "TODO",
	})
	h.store.ClearFlow(chatID)

	if err != nil {
		h.sendText(chatID, "❌ Failed to create: "+err.Error())
		return
	}

	h.sendHTML(chatID, fmt.Sprintf("✅ Task created!\n\n<b>%s</b>\nID: <code>%s</code>", title, task.ID))
}
