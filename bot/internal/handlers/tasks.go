package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleTasks(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "TODO")
	if err != nil {
		h.editOrSend(chatID, messageID,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.", keyboards.HomeInline())
		return
	}

	if len(tasks) == 0 {
		h.editOrSend(chatID, messageID, "📋 <b>Задач нет</b>", keyboards.TaskListEmpty())
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
	h.editOrSend(chatID, messageID, text, kb)
}

func (h *Handler) handleTaskAction(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.editOrSend(chatID, messageID, "❌ Не удалось загрузить задачу.", keyboards.BackToTasks())
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
		h.editOrSend(chatID, messageID, "Задача не найдена.", keyboards.BackToTasks())
		return
	}

	text := fmt.Sprintf("%s <b>%s</b>\n", format.PriorityEmoji(priority), format.Escape(title))
	if estMin > 0 {
		text += fmt.Sprintf("⏱ %s\n", format.Duration(estMin))
	}
	if dueDate != "" {
		text += fmt.Sprintf("📅 Due: %s\n", format.FormatDate(dueDate, h.cfg.Timezone))
	}

	h.editOrSend(chatID, messageID, text, keyboards.TaskActions(taskID))
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

// Срок / Оценка / Теги — the three fields the card could show but never let
// you set. Due and estimate are button-driven, and reuse keyboards.TaskDue /
// TaskEstimate (which themselves reuse the wizard's own dueRow/estimateRow —
// see keyboards.go). Tags are free text: there is no closed set of values to
// offer, so "🏷 Теги" opens a normal FlowData text prompt, same shape as
// startNoteFlow / startNewTaskFlow.

var dueOffsets = map[string]bool{"0": true, "1": true, "7": true}
var estimateOptions = map[string]bool{"15": true, "30": true, "60": true, "120": true}

func (h *Handler) handleTaskDueMenu(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	h.editOrSend(chatID, messageID,
		fmt.Sprintf("📅 <b>%s</b>\n\nКогда срок?", format.Escape(title)),
		keyboards.TaskDue(taskID))
}

// handleTaskDueSet answers task_due_set_<uuid>_<offset>, cut from the right —
// same convention as parsePlanCallback in schedule.go, for the same reason: a
// task id is opaque to this bot and the offset is ours and known-shaped.
func (h *Handler) handleTaskDueSet(chatID int64, messageID int, data string) {
	idx := strings.LastIndex(data, "_")
	if idx < 0 {
		h.sendHTMLWithKeyboard(chatID, "Не понял кнопку.", keyboards.BackToTasks())
		return
	}
	taskID, offsetStr := data[:idx], data[idx+1:]
	if taskID == "" || !dueOffsets[offsetStr] {
		h.sendHTMLWithKeyboard(chatID, "Не понял кнопку.", keyboards.BackToTasks())
		return
	}
	offset, _ := strconv.Atoi(offsetStr)

	due := time.Now().In(h.location()).AddDate(0, 0, offset)
	us := h.store.GetOrCreate(chatID)
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{
		"due_date": due.Format(time.RFC3339),
	}); err != nil {
		h.sendText(chatID, "❌ Не удалось сохранить: "+err.Error())
		return
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

func (h *Handler) handleTaskEstimateMenu(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	h.editOrSend(chatID, messageID,
		fmt.Sprintf("⏱ <b>%s</b>\n\nСколько времени займёт?", format.Escape(title)),
		keyboards.TaskEstimate(taskID))
}

// handleTaskEstimateSet answers task_est_set_<uuid>_<minutes>.
func (h *Handler) handleTaskEstimateSet(chatID int64, messageID int, data string) {
	idx := strings.LastIndex(data, "_")
	if idx < 0 {
		h.sendHTMLWithKeyboard(chatID, "Не понял кнопку.", keyboards.BackToTasks())
		return
	}
	taskID, minStr := data[:idx], data[idx+1:]
	if taskID == "" || !estimateOptions[minStr] {
		h.sendHTMLWithKeyboard(chatID, "Не понял кнопку.", keyboards.BackToTasks())
		return
	}
	minutes, _ := strconv.Atoi(minStr)

	us := h.store.GetOrCreate(chatID)
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{
		"estimated_minutes": minutes,
	}); err != nil {
		h.sendText(chatID, "❌ Не удалось сохранить: "+err.Error())
		return
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

// handleTaskTagsPrompt answers task_tag_<uuid> — "🏷 Теги" on the card. It
// opens a text flow rather than a keyboard: tags are not a closed set.
func (h *Handler) handleTaskTagsPrompt(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "edit_task_tags"
	us.FlowStep = "text"
	us.FlowData["taskID"] = taskID
	h.editOrSend(chatID, messageID,
		fmt.Sprintf("🏷 <b>%s</b>\n\nТэги через запятую (или «cancel»):", format.Escape(title)),
		tgbotapi.NewInlineKeyboardMarkup())
}

// handleEditTaskTags is the text-flow answer to handleTaskTagsPrompt, routed
// from handleFlowInput (flows.go) by CurrentFlow == "edit_task_tags".
//
// The project rule is empty slices, never nil: typing "-" or a blank line
// clears the tags rather than leaving the field untouched, and it does so by
// sending an explicit [] — omitting the key here would mean "don't change
// this field", which is not what a user asking to clear their tags wants.
func (h *Handler) handleEditTaskTags(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	taskID, _ := us.FlowData["taskID"].(string)
	h.store.ClearFlow(chatID)
	if taskID == "" {
		h.sendHTMLWithKeyboard(chatID, "Не помню, к какой задаче это относится.", keyboards.HomeInline())
		return
	}

	tags := []string{}
	for _, tag := range strings.Split(text, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" && tag != "-" {
			tags = append(tags, tag)
		}
	}

	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"tags": tags}); err != nil {
		h.sendText(chatID, "❌ Не удалось сохранить: "+err.Error())
		return
	}
	h.sendText(chatID, "🏷 Тэги обновлены")
	// 0: the tags arrived as a text message, so there is no screen of ours
	// under the user thumb to edit — the card is posted fresh.
	h.handleTaskAction(chatID, 0, taskID)
}
