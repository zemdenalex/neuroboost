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

	// 🔴 Denis, 23.09: «seamless task creation» — a line without a command IS
	// a task, saved at once; the kind question moves to buttons under the
	// saved task (quicksave.go). Only lines where a question is genuinely
	// needed go on to the older path below.
	if r, ok := h.plainTaskLine(chatID, text); ok {
		h.quickSaveTask(chatID, text, r)
		return
	}

	// 🔴 Denis, 17.09 (second pass): «надо чтобы слово задача сделала на одно
	// действие меньше, то есть бот воспринял как задачу, но уточнил, а не
	// заставлял выбирать задачу еще раз». The word answers the question; the
	// card that follows still offers the other kinds.
	if parse.IsTaskLine(text) {
		us.FlowData = map[string]any{"raw": text}
		if parse.ParseLine(text, time.Now().In(h.location(chatID))).Draft.HasTime {
			h.quickAs(chatID, "event", text)
		} else {
			h.quickAs(chatID, "task", text)
		}
		return
	}
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

	// The line is what these buttons act on — it is remembered through every
	// flow quick add hands over to, so «сделать событием» works from the task
	// card too. Without it there is nothing to act on.
	if raw == "" {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Не помню, о чём это было, напиши ещё раз.", "I no longer remember what this was about; write it again."),
			keyboards.HomeInline(h.lang(chatID)))
		return true
	}

	switch data {
	case "qa_event":
		// Chosen as an event: «задача» in the line is a word, not an order.
		h.quickKind = kindEvent
		h.quickAs(chatID, "event", raw)
	case "qa_task":
		// A task with a clock time is a task bound to an event — only the
		// event card builds both. Without a time it is a plain task.
		if parse.ParseLine(raw, time.Now().In(h.location(chatID))).Draft.HasTime {
			h.quickKind = kindTask
			h.quickAs(chatID, "event", raw)
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
	raw, _ := us.FlowData["raw"].(string)
	if raw == "" {
		raw = text
	}
	// The line is kept so the card can offer the OTHER kinds without asking
	// the user to type it again.
	us.FlowData = map[string]any{"raw": raw}
	switch kind {
	case "event":
		us.CurrentFlow, us.FlowStep = "new_event", "line"
		us.FlowData["raw"] = raw
		h.handleNewEventFlow(chatID, text)
	case "task":
		us.CurrentFlow, us.FlowStep = "new_task", "title"
		us.FlowData["raw"] = raw
		h.handleNewTaskFlow(chatID, text)
	case "note":
		us.CurrentFlow, us.FlowStep = "note", "text"
		h.handleNoteFlow(chatID, text)
	}
}

// The kind a qa_ button chose, applied to the draft before the card is drawn.
//
// 🔴 It used to be applied AFTER — draw the card, change the kind, draw again —
// and Denis got two cards for one press (his log, 17.09 21:56).
const (
	kindUnset = ""
	kindEvent = "event"
	kindTask  = "task"
)

// applyQuickKind writes the chosen kind into a freshly parsed draft.
func (h *Handler) applyQuickKind(st *draftState) {
	switch h.quickKind {
	case kindEvent:
		st.D.IsTask = false
	case kindTask:
		st.D.IsTask = true
	}
	h.quickKind = kindUnset
}
