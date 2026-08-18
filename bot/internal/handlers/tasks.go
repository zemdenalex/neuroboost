package handlers

import (
	"fmt"
	"sort"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleTasks(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "TODO")
	if err != nil {
		h.sendText(chatID, "❌ Failed to load tasks: "+err.Error())
		return
	}

	if len(tasks) == 0 {
		h.sendHTMLWithKeyboard(chatID, "📋 <b>No tasks</b>\n\nUse ➕ New Task to create one.", keyboards.BackToMenu())
		return
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Priority < tasks[j].Priority })

	text := fmt.Sprintf("📋 <b>Tasks (%d)</b>\n\n", len(tasks))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, t := range tasks {
		if i >= 10 {
			break
		}
		label := fmt.Sprintf("%s %s", format.PriorityEmoji(t.Priority), t.Title)
		if len(label) > 40 {
			label = label[:37] + "..."
		}
		text += fmt.Sprintf("%s %s\n", format.PriorityEmoji(t.Priority), format.Escape(t.Title))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "task_action_"+t.ID),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.sendHTMLWithKeyboard(chatID, text, kb)
}

func (h *Handler) handleTaskAction(chatID int64, taskID string) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.sendText(chatID, "❌ Failed to load task")
		return
	}

	var found bool
	var title string
	var priority, estMin int
	var dueDate string
	for _, t := range tasks {
		if t.ID == taskID {
			found = true
			title = t.Title
			priority = t.Priority
			estMin = t.EstimatedMinutes
			dueDate = t.DueDate
			break
		}
	}

	if !found {
		h.sendText(chatID, "Task not found")
		return
	}

	text := fmt.Sprintf("%s <b>%s</b>\n", format.PriorityEmoji(priority), format.Escape(title))
	if estMin > 0 {
		text += fmt.Sprintf("⏱ %s\n", format.Duration(estMin))
	}
	if dueDate != "" {
		text += fmt.Sprintf("📅 Due: %s\n", format.FormatDate(dueDate, h.cfg.Timezone))
	}

	h.sendHTMLWithKeyboard(chatID, text, keyboards.TaskActions(taskID))
}

func (h *Handler) handleTaskDone(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"status": "DONE"})
	if err != nil {
		h.sendText(chatID, "❌ Failed: "+err.Error())
		return
	}
	h.sendText(chatID, "✅ Task completed!")
	h.handleTasks(chatID, messageID)
}

func (h *Handler) handleTaskDelete(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	err := h.api.DeleteTask(us.AuthToken, taskID)
	if err != nil {
		h.sendText(chatID, "❌ Failed: "+err.Error())
		return
	}
	h.sendText(chatID, "🗑 Task deleted")
	h.handleTasks(chatID, messageID)
}
