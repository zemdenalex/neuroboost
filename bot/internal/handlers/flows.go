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

func (h *Handler) startNoteFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "note"
	us.FlowStep = "text"
	h.sendText(chatID, "📝 Пришли заметку — сохраню её задачей.")
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
	case "keyword":
		h.handleKeywordInput(chatID, text)
	case "edit_task_tags":
		h.handleEditTaskTags(chatID, text)
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
		h.sendText(chatID, "❌ Не удалось сохранить: "+err.Error())
		return
	}
	h.sendText(chatID, "✅ Заметка сохранена задачей.")
}

func (h *Handler) handleNewTaskFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	switch us.FlowStep {
	case "title":
		// 🔴 Ask before assuming. Denis, 15.09: «С задачами ты сделал тоже
		// списки?» — no, and three lines silently became one task carrying a
		// three-line title.
		if parse.LooksLikeList(text, time.Now().In(h.location())) {
			us.FlowData["raw"] = text
			us.FlowStep = "list:confirm"
			n := len(parse.Entries(text))
			h.sendHTMLWithKeyboard(chatID,
				fmt.Sprintf("Это одна задача или список из %d?\n\n<i>Одной задачей название будет целиком, со всеми строками.</i>", n),
				keyboards.ListConfirm(n))
			return
		}
		h.showTaskCard(chatID, text)
	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.HomeInline())
	}
}

// handleTaskCardSave creates the task from the card's understanding as it
// stands — nothing more is asked, because nothing more is required.
//
// 🔴 It answers "nt_save" — and "✅ Создать сейчас" (keyboards.wizardEscapes)
// sends that exact callback from every wizard screen, not just the plain
// card. The guard used to require FlowStep == "card" and nothing else, so
// nt_save pressed from "wizard:priority"/"wizard:due"/"wizard:estimate" was
// silently swallowed here: the callback got answered (spinner stops), and
// nothing else happened. That was the one escape Denis named by hand
// ("должна быть кнопка ... создать сейчас"), and it was the one that did
// nothing. A step's own value, if any, is still in FlowData at that point —
// advanceWizard's "done" path proves that already — so saving from here is
// exactly as complete as saving from the card.
func (h *Handler) handleTaskCardSave(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	onWizardStep := strings.HasPrefix(us.FlowStep, "wizard:")
	if us.CurrentFlow != "new_task" || (us.FlowStep != "card" && !onWizardStep) {
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
