package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Event → task (spec 21.09 §A2). The card is the API's dry run: the mapping —
// which day, how long, what is lost — is computed once, in api-go, in the
// user's zone, and the bot only words it.

const toTaskFlow = "to_task"

// handleToTaskStart answers «✅ Сделать задачей» on an event.
func (h *Handler) handleToTaskStart(chatID int64, messageID int, rawEventID string) {
	us := h.store.GetOrCreate(chatID)
	parentID, _, _ := splitInstanceID(rawEventID)
	ev, err := h.api.GetEvent(us.AuthToken, parentID)
	if err != nil || ev == nil || ev.ID == "" {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"❌ Не удалось открыть событие — возможно, оно уже удалено.",
			"❌ Could not open the event — it may already be deleted."),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	us.CurrentFlow = toTaskFlow
	us.FlowStep = "how"
	us.FlowData = map[string]any{"eventID": rawEventID, "title": ev.Title,
		"repeats": ev.Rrule != nil && *ev.Rrule != ""}
	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
		"✅ <b>%s</b> — сделать задачей\n\n🔗 <b>Связать</b> — событие останется и будет указывать на задачу.\n➡️ <b>Перенести</b> — останется только задача.",
		"✅ <b>%s</b> — make it a task\n\n🔗 <b>Link</b> — the event stays and points at the task.\n➡️ <b>Move</b> — only the task remains."),
		format.Escape(ev.Title)), keyboards.ConvertHow(h.lang(chatID), "e2"))
}

// handleToTaskStep is every e2* button after the first.
func (h *Handler) handleToTaskStep(chatID int64, messageID int, data string) {
	us := h.store.GetOrCreate(chatID)
	if data == "e2x" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID, "Отменено.", "Cancelled."), keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	if data == "e2i" {
		h.sendHTML(chatID, convertExplanation(h.lang(chatID)))
		return
	}
	if us.CurrentFlow != toTaskFlow {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Это меню устарело — открой событие заново.", "This menu is out of date — open the event again."),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	rawID, _ := us.FlowData["eventID"].(string)
	_, _, isInstance := splitInstanceID(rawID)
	repeats, _ := us.FlowData["repeats"].(bool)

	switch data {
	case "e2m_l", "e2m_m":
		us.FlowData["mode"] = map[string]string{"e2m_l": "link", "e2m_m": "move"}[data]
		if repeats {
			text := h.t(chatID,
				"🔁 Это повторяющееся событие.\n\n<b>Вся серия</b> — задача тоже будет повторяться.",
				"🔁 This event repeats.\n\n<b>Whole series</b> — the task repeats too.")
			if isInstance {
				text += h.t(chatID, "\n<b>Только этот раз</b> — задача на один день, серия событий останется.",
					"\n<b>Just this once</b> — a task for one day; the series of events stays.")
			} else {
				text += h.t(chatID, "\n\n⚠ Открыта вся серия, поэтому один день выбрать нельзя — открой событие из дня календаря.",
					"\n\n⚠ The whole series is open, so one day cannot be chosen — open the event from a calendar day.")
			}
			h.editOrSend(chatID, messageID, text, keyboards.ConvertRepeat(h.lang(chatID), "e2", isInstance))
			return
		}
		h.showToTaskCard(chatID, messageID)
	case "e2r_s", "e2r_o":
		if data == "e2r_o" && !isInstance {
			return // not offered; a crafted button changes nothing
		}
		us.FlowData["repeat"] = map[string]string{"e2r_s": "series", "e2r_o": "once"}[data]
		h.showToTaskCard(chatID, messageID)
	case "e2ok":
		h.finishToTask(chatID, messageID)
	}
}

func (h *Handler) toTaskReq(chatID int64, dry bool) (string, api.ToTaskReq) {
	us := h.store.GetOrCreate(chatID)
	rawID, _ := us.FlowData["eventID"].(string)
	mode, _ := us.FlowData["mode"].(string)
	repeat, _ := us.FlowData["repeat"].(string)
	return rawID, api.ToTaskReq{Mode: mode, Repeat: repeat, DryRun: dry}
}

func (h *Handler) showToTaskCard(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	rawID, req := h.toTaskReq(chatID, true)
	res, err := h.api.EventToTask(us.AuthToken, rawID, req)
	if err != nil {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не получилось: ", "❌ Did not work: ")+format.Escape(h.errorText(chatID, err)),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	title, _ := us.FlowData["title"].(string)
	h.editOrSend(chatID, messageID, toTaskCard(h.lang(chatID), title, *res, req.Mode, req.Repeat),
		keyboards.ConvertConfirm(h.lang(chatID), "e2"))
}

func (h *Handler) finishToTask(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	rawID, req := h.toTaskReq(chatID, false)
	if req.Mode == "" {
		h.store.ClearFlow(chatID)
		return
	}
	res, err := h.api.EventToTask(us.AuthToken, rawID, req)
	h.store.ClearFlow(chatID)
	if err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не получилось: ", "❌ Did not work: ")+format.Escape(h.errorText(chatID, err)),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	// The new task's own card: the proof it exists, and everything to do next.
	h.handleTaskAction(chatID, messageID, res.Task.ID)
}

// toTaskCard words the dry run. Pure, so every line is tested.
func toTaskCard(lang i18n.Lang, title string, r api.ToTaskResult, mode, repeat string) string {
	var b strings.Builder
	fmt.Fprintf(&b, i18n.T(lang, "✅ <b>%s</b> → задача\n\n", "✅ <b>%s</b> → task\n\n"), format.Escape(title))
	if due, err := time.Parse("2006-01-02", r.Task.DueDate); err == nil {
		fmt.Fprintf(&b, i18n.T(lang, "срок: %s\n", "due: %s\n"), dayLabel(lang, due))
	}
	if r.Task.EstimatedMinutes != nil {
		fmt.Fprintf(&b, i18n.T(lang, "оценка: %s\n", "estimate: %s\n"), format.Duration(*r.Task.EstimatedMinutes))
	}
	b.WriteString(i18n.T(lang, "описание, теги → как есть\n", "description, tags → as they are\n"))
	if r.Task.Rrule != nil {
		b.WriteString(i18n.T(lang, "🔁 повтор → задача повторяется\n", "🔁 repeat → the task repeats\n"))
	}
	for _, code := range r.Lost {
		b.WriteString(lostLine(lang, code) + "\n")
	}
	switch {
	case mode == "link":
		b.WriteString(i18n.T(lang, "\n🔗 Событие останется и будет указывать на задачу.", "\n🔗 The event stays and points at the task."))
	case repeat == "once":
		b.WriteString(i18n.T(lang, "\n➡️ Этот день уйдёт из серии, серия событий останется.", "\n➡️ This day leaves the series; the series stays."))
	default:
		b.WriteString(i18n.T(lang, "\n➡️ Событие удалится, останется задача.", "\n➡️ The event is deleted; the task remains."))
	}
	return b.String()
}

// lostLine words a lost-field code from api-go (events.Lost*).
func lostLine(lang i18n.Lang, code string) string {
	switch code {
	case "start_time":
		return i18n.T(lang, "⚠ час начала → у задачи нет", "⚠ start hour → tasks have none")
	case "color":
		return i18n.T(lang, "⚠ цвет → у задачи нет", "⚠ colour → tasks have none")
	case "location":
		return i18n.T(lang, "⚠ место → у задачи нет", "⚠ place → tasks have none")
	case "reminders":
		return i18n.T(lang, "⚠ напоминания → у задачи будут свои, по умолчанию", "⚠ reminders → the task gets your default ones")
	default:
		return i18n.T(lang, "⚠ ещё кое-что не переносится", "⚠ something else does not carry over")
	}
}
