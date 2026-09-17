package handlers

import (
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// Quick add — a line typed without pressing anything first.
//
// 🔴 Denis, 17.09: «сейчас когда пишешь что-то без предыдущей команды, он
// просто говорит, что не понял, а я хочу, чтобы он спрашивал, создать новое
// событие, задачу, заметку или что-то ещё… и использовал все свои функции».
//
// The last clause is the design. Quick add has no parser of its own: after the
// one question it hands the text to the SAME flow the menu opens, so the card,
// the list question, reminders, keywords and calendars all apply unchanged. A
// second, lighter path would be a second dialect that drifts.

const quickFlow = "quick"

func (h *Handler) handleQuickAdd(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = quickFlow
	us.FlowStep = "kind"
	us.FlowData = map[string]any{"raw": text}

	// The line is quoted back so the choice is made about something visible —
	// in a chat, the message being asked about may already be off screen.
	h.sendHTMLWithKeyboard(chatID,
		"<i>"+format.Escape(text)+"</i>\n\n"+h.t(chatID, "Что создать?", "What should I create?"),
		// 🔴 «задача» does not skip the question (Denis, 17.09: «даже если мы
		// пишем задача, надо чтобы был выбор сохранить как событие или
		// заметку»). It only puts Task first.
		keyboards.QuickAddKind(h.lang(chatID), parse.IsTaskLine(text)))
}

// handleQuickCallback answers the qa_ buttons. Returns false for anything else.
func (h *Handler) handleQuickCallback(chatID int64, messageID int, data string) bool {
	if !strings.HasPrefix(data, "qa_") {
		return false
	}
	us := h.store.GetOrCreate(chatID)
	raw, _ := us.FlowData["raw"].(string)

	if data == "qa_cancel" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID, "Отменено.", "Cancelled."), keyboards.HomeInline(h.lang(chatID)))
		return true
	}

	// A button under an old question, after the flow moved on: the text it
	// referred to is gone, and guessing which line it meant would be worse.
	if us.CurrentFlow != quickFlow || raw == "" {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Не помню, о чём это было — напиши ещё раз.", "I no longer remember what this was about — write it again."),
			keyboards.HomeInline(h.lang(chatID)))
		return true
	}

	switch data {
	case "qa_event":
		h.quickAs(chatID, "event", raw)
		// Chosen as an event: «задача» in the line is a word, not an order.
		if st, ok := draftOf(h, chatID); ok && st.D.IsTask {
			st.D.IsTask = false
			h.showCard(chatID, 0)
		}
	case "qa_task":
		// A task with a clock time is a task bound to an event — only the
		// event card builds both. Without a time it is a plain task.
		if parse.ParseLine(raw, time.Now().In(h.location(chatID))).Draft.HasTime {
			h.quickAs(chatID, "event", raw)
			if st, ok := draftOf(h, chatID); ok && !st.D.IsTask {
				st.D.IsTask = true
				h.showCard(chatID, 0)
			}
		} else {
			h.quickAs(chatID, "task", raw)
		}
	case "qa_note":
		h.quickAs(chatID, "note", raw)
	default:
		return false
	}
	return true
}

// quickAs enters a creation flow exactly as its menu button would, then feeds
// it the line — skipping only the guide, which this user has no need to read.
func (h *Handler) quickAs(chatID int64, kind, text string) {
	us := h.store.GetOrCreate(chatID)
	us.FlowData = map[string]any{}
	switch kind {
	case "event":
		us.CurrentFlow, us.FlowStep = "new_event", "line"
		h.handleNewEventFlow(chatID, text)
	case "task":
		us.CurrentFlow, us.FlowStep = "new_task", "title"
		h.handleNewTaskFlow(chatID, text)
	case "note":
		us.CurrentFlow, us.FlowStep = "note", "text"
		h.handleNoteFlow(chatID, text)
	}
}
