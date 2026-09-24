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
	h.sendText(chatID, h.t(chatID, "📝 Пришли заметку, сохраню её задачей.", "📝 Send a note; I'll save it as a task."))
}

func (h *Handler) handleFlowInput(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	if strings.ToLower(text) == "cancel" || text == "/start" {
		h.store.ClearFlow(chatID)
		h.handleStart(chatID)
		return
	}

	// The calendar screens carry their target in the flow name («cal:name:<id>»),
	// so they are matched by prefix rather than listed one case per action.
	if strings.HasPrefix(us.CurrentFlow, calendarFlowPrefix) {
		h.handleCalendarText(chatID, us.CurrentFlow, text)
		return
	}

	if strings.HasPrefix(us.CurrentFlow, feedbackFlowPrefix) {
		h.handleFeedbackText(chatID, us.CurrentFlow, text)
		return
	}

	if strings.HasPrefix(us.CurrentFlow, snoozeFlowPrefix) {
		h.handleSnoozeText(chatID, us.CurrentFlow, text)
		return
	}

	switch us.CurrentFlow {
	case onboardFlow:
		h.handleOnboardText(chatID, text)
	case quickFlow:
		// A new line while the question is still open replaces the old one:
		// the latest thing typed is what the user wants to create.
		h.handleQuickAdd(chatID, text)
	case "note":
		h.handleNoteFlow(chatID, text)
	case dayTaskDateFlow:
		h.handleDayTaskDate(chatID, text)
	case "new_task":
		h.handleNewTaskFlow(chatID, text)
	case "new_event":
		h.handleNewEventFlow(chatID, text)
	case "keyword":
		h.handleKeywordInput(chatID, text)
	case "postpone_custom":
		h.handlePostponeCustomText(chatID, text)
	case toEventFlow:
		h.handleToEventText(chatID, text)
	case "edit_task_tags":
		h.handleEditTaskTags(chatID, text)
	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Что-то пошло не так.", "Something went wrong."), h.home(chatID))
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
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	h.sendText(chatID, h.t(chatID, "✅ Заметка сохранена задачей.", "✅ Note saved as a task."))
}

func (h *Handler) handleNewTaskFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	switch us.FlowStep {
	case "title":
		// 🔴 Ask before assuming. Denis, 15.09: «С задачами ты сделал тоже
		// списки?» — no, and three lines silently became one task carrying a
		// three-line title.
		if parse.LooksLikeList(text, time.Now().In(h.location(chatID))) {
			us.FlowData["raw"] = text
			us.FlowStep = "list:confirm"
			n := 0
			for _, task := range parse.ParseTaskList(text, time.Now().In(h.location(chatID))) {
				if strings.TrimSpace(task.Title) != "" {
					n++
				}
			}
			h.sendHTMLWithKeyboard(chatID,
				fmt.Sprintf(h.t(chatID, "Это одна задача или список из %d?\n\n<i>Одной задачей название будет целиком, со всеми строками.</i>", "One task, or a list of %d?\n\n<i>As one task the title keeps every line.</i>"), n),
				keyboards.ListConfirm(h.lang(chatID), n))
			return
		}
		h.showTaskCard(chatID, text)
	case "card":
		// 🔴 Denis, 18.09: «если начать создавать например задачу, и написать
		// другую то он пишет что-то пошло не так». A line typed while a card is
		// on screen is a NEW thing, not a malformed answer — the card asked no
		// question. Same rule as list:confirm below and as the event card: the
		// latest line replaces what is shown.
		us.FlowStep = "title"
		us.FlowData["raw"] = text
		h.handleNewTaskFlow(chatID, text)

	case "list:confirm":
		// 🔴 Typing here used to answer «Что-то пошло не так» and throw the
		// list away (Denis, 17.09: «А список вообще не понял»). A new line
		// replaces the question.
		us.FlowStep = "title"
		h.handleNewTaskFlow(chatID, text)
	default:
		// «✏️ Переписать» on one entry of a task list.
		if strings.HasPrefix(us.FlowStep, "list:rewrite:") {
			h.rewriteTaskListEntry(chatID, text)
			return
		}
		// A wizard step that can be answered in words is answered in words.
		if strings.HasPrefix(us.FlowStep, "wizard:") && h.handleWizardText(chatID, text) {
			return
		}
		if us.FlowStep == "list" || strings.HasPrefix(us.FlowStep, "wizard:") {
			// A screen that wants a button: the list survives.
			h.sendHTMLWithKeyboard(chatID, h.t(chatID,
				"Здесь нужна кнопка, список на месте.",
				"This screen needs a button; the list is still here."), keyboards.BackToTasks(h.lang(chatID)))
			return
		}
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Что-то пошло не так.", "Something went wrong."), h.home(chatID))
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
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не помню название.", "I've lost the title."), h.home(chatID))
		return
	}

	// «⚠ повтор — частота не указана, спрошу» — and now it does.
	//
	// 🔴 21.09, Denis, next to that very line: «Ну пишет спрошу, но создать
	// даёт без спроса». The card has said this since 18.09 and nothing ever
	// asked. A promise printed and not kept is worse than no promise: it
	// teaches that the card's lines are decoration, and the card is the only
	// thing standing between a typed sentence and a row in the database.
	//
	// ⚠ The flag is dropped as the question goes up, not when it is answered.
	// The promise was to ask ONCE — skipping the step, or pressing ✅ again,
	// must move on rather than loop. That loop is what a «refuse until
	// answered» reading of this would build.
	if asked, _ := us.FlowData["repeat_asked"].(bool); asked {
		delete(us.FlowData, "repeat_asked")
		if rule, ok := us.FlowData["rrule"].(string); !ok || rule == "" {
			// Marked so the answer comes straight back here instead of walking
			// on into the rest of the wizard — see advanceWizard.
			us.FlowData["repeat_from_card"] = true
			h.showWizardStep(chatID, messageID, "repeat")
			return
		}
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
	if rule, ok := us.FlowData["rrule"].(string); ok && rule != "" {
		req.Rrule = &rule
	}

	task, err := h.api.CreateTask(us.AuthToken, req)
	h.store.ClearFlow(chatID)
	if err != nil {
		h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не удалось создать: ", "❌ Could not create: ")+h.errorText(chatID, err), h.home(chatID))
		return
	}

	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "✅ <b>Задача создана</b>\n%s", "✅ <b>Task created</b>\n%s"), format.Escape(task.Title)),
		h.home(chatID))
}
