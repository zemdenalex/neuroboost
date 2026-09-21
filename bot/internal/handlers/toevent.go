package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// Task → event, the full path (spec 21.09 §A2). Denis's shape «1 + 3»: ask
// move or link · show what becomes what · ask only what is missing. The quick
// «⏰ Запланировать» stays two taps; this is the path that explains itself.

const toEventFlow = "to_event"

// toEventDurations is the closed set t2d_ accepts; callback data is user input.
var toEventDurations = map[int]bool{15: true, 30: true, 60: true, 120: true}

// handleToEventStart answers «📅 В календарь» on the task card.
func (h *Handler) handleToEventStart(chatID int64, messageID int, taskID string) {
	task, ok := h.taskByID(chatID, taskID)
	if !ok {
		return
	}
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = toEventFlow
	us.FlowStep = "how"
	us.FlowData = map[string]any{"taskID": taskID}
	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
		"📅 <b>%s</b> — в календарь\n\n🔗 <b>Связать</b> — задача останется в списке, у события будут её время и напоминания.\n➡️ <b>Перенести</b> — останется только событие.",
		"📅 <b>%s</b> — to the calendar\n\n🔗 <b>Link</b> — the task stays in the list; the event carries its time and reminders.\n➡️ <b>Move</b> — only the event remains."),
		format.Escape(task.Title)), keyboards.ConvertHow(h.lang(chatID), "t2"))
}

// handleToEventStep is every t2* button after the first.
func (h *Handler) handleToEventStep(chatID int64, messageID int, data string) {
	us := h.store.GetOrCreate(chatID)
	if data == "t2x" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID, "Отменено.", "Cancelled."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	if data == "t2i" {
		h.sendHTML(chatID, convertExplanation(h.lang(chatID)))
		return
	}
	if us.CurrentFlow != toEventFlow {
		// A button from a flow that is gone: say so, write nothing.
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Это меню устарело — открой задачу заново.", "This menu is out of date — open the task again."),
			keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	taskID, _ := us.FlowData["taskID"].(string)
	task, ok := h.taskByID(chatID, taskID)
	if !ok {
		h.store.ClearFlow(chatID)
		return
	}

	switch {
	case data == "t2m_l" || data == "t2m_m":
		us.FlowData["mode"] = map[string]string{"t2m_l": "link", "t2m_m": "move"}[data]
		if task.Repeats() {
			h.editOrSend(chatID, messageID, h.t(chatID,
				"🔁 Это повторяющаяся задача.\n\n<b>Вся серия</b> — событие тоже будет повторяться.\n<b>Только этот раз</b> — в календарь уйдёт один день, серия задачи останется.",
				"🔁 This task repeats.\n\n<b>Whole series</b> — the event repeats too.\n<b>Just this once</b> — one day goes to the calendar; the series stays a task."),
				keyboards.ConvertRepeat(h.lang(chatID), "t2", true))
			return
		}
		h.askToEventWhen(chatID, messageID)
	case data == "t2r_s" || data == "t2r_o":
		us.FlowData["repeat"] = map[string]string{"t2r_s": "series", "t2r_o": "once"}[data]
		h.askToEventWhen(chatID, messageID)
	case strings.HasPrefix(data, "t2w_"):
		slot := strings.TrimPrefix(data, "t2w_")
		start, resolved := scheduleStart(slot, time.Now(), h.location(chatID))
		if !scheduleSlots[slot] || !resolved {
			return
		}
		h.toEventHaveStart(chatID, messageID, task, start)
	case data == "t2c":
		us.FlowStep = "time_text"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"✏️ Напиши день и время: «завтра 15:00», «пт 10:30». Или «cancel».",
			"✏️ Type a day and time: «tomorrow 15:00», «fri 10:30». Or «cancel»."),
			tgbotapi.NewInlineKeyboardMarkup(keyboards.ConvertCancel(h.lang(chatID), "t2")))
	case strings.HasPrefix(data, "t2d_"):
		minutes, err := strconv.Atoi(strings.TrimPrefix(data, "t2d_"))
		if err != nil || !toEventDurations[minutes] {
			return
		}
		us.FlowData["minutes"] = minutes
		h.showToEventCard(chatID, messageID, task)
	case data == "t2ok":
		h.finishToEvent(chatID, messageID, task)
	}
}

func (h *Handler) askToEventWhen(chatID int64, messageID int) {
	h.editOrSend(chatID, messageID, h.t(chatID, "🕐 Когда?", "🕐 When?"), keyboards.ToEventWhen(h.lang(chatID)))
}

// handleToEventText is the «✏️ Своё» answer.
func (h *Handler) handleToEventText(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	taskID, _ := us.FlowData["taskID"].(string)
	task, ok := h.taskByID(chatID, taskID)
	if !ok {
		h.store.ClearFlow(chatID)
		return
	}
	loc := h.location(chatID)
	now := time.Now().In(loc)
	p := parse.ParseLine(text, now)
	if !p.Draft.HasTime {
		// 🔴 A day without an hour is asked again, never completed with a
		// guess — the flow stays open, the same rule the wizard follows.
		h.sendText(chatID, h.t(chatID,
			"Нужно и время, например «завтра 15:00». Или «cancel».",
			"I need a time too, e.g. «tomorrow 15:00». Or «cancel»."))
		return
	}
	day := p.Draft.Day
	if !p.Draft.HasDay {
		day = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
	us.FlowStep = "card"
	h.toEventHaveStart(chatID, 0, task, day.Add(p.Draft.Start))
}

func (h *Handler) toEventHaveStart(chatID int64, messageID int, task api.Task, start time.Time) {
	us := h.store.GetOrCreate(chatID)
	us.FlowData["start"] = start.Format(time.RFC3339)
	if task.EstimatedMinutes > 0 {
		us.FlowData["minutes"] = task.EstimatedMinutes
		h.showToEventCard(chatID, messageID, task)
		return
	}
	h.editOrSend(chatID, messageID, h.t(chatID,
		"⏱ Насколько? У задачи нет оценки.", "⏱ How long? The task has no estimate."),
		keyboards.ToEventDuration(h.lang(chatID)))
}

func (h *Handler) showToEventCard(chatID int64, messageID int, task api.Task) {
	us := h.store.GetOrCreate(chatID)
	start, _ := time.Parse(time.RFC3339, fmt.Sprint(us.FlowData["start"]))
	minutes, _ := us.FlowData["minutes"].(int)
	mode, _ := us.FlowData["mode"].(string)
	repeat, _ := us.FlowData["repeat"].(string)
	now := time.Now().In(h.location(chatID))
	h.editOrSend(chatID, messageID,
		toEventCard(h.lang(chatID), task, mode, repeat, start.In(now.Location()), minutes, now),
		keyboards.ConvertConfirm(h.lang(chatID), "t2"))
}

// toEventCard is the «что чем станет» screen. Pure, so its lines are tested.
func toEventCard(lang i18n.Lang, t api.Task, mode, repeat string, start time.Time, minutes int, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, i18n.T(lang, "📅 <b>%s</b> → событие\n\n", "📅 <b>%s</b> → event\n\n"), format.Escape(t.Title))
	b.WriteString(i18n.T(lang, "название → название\n", "title → title\n"))
	fmt.Fprintf(&b, i18n.T(lang, "длительность: %s\n", "length: %s\n"), format.Duration(minutes))
	fmt.Fprintf(&b, i18n.T(lang, "когда: %s\n", "when: %s\n"), whenShort(lang, start, now))
	b.WriteString(i18n.T(lang, "описание, теги → как есть\n", "description, tags → as they are\n"))
	switch {
	case t.Repeats() && repeat == "series":
		b.WriteString(i18n.T(lang, "🔁 повтор → событие повторяется\n", "🔁 repeat → the event repeats\n"))
	case t.Repeats():
		b.WriteString(i18n.T(lang, "1️⃣ в календарь уходит один день\n", "1️⃣ one day goes to the calendar\n"))
	}
	b.WriteString(i18n.T(lang, "⚠ приоритет → у события нет\n", "⚠ priority → events have none\n"))
	switch {
	case mode == "link":
		b.WriteString(i18n.T(lang, "\n🔗 Задача останется в списке с пометкой времени.", "\n🔗 The task stays in the list, marked with the time."))
	case t.Repeats() && repeat == "once":
		b.WriteString(i18n.T(lang, "\n➡️ Этот день уйдёт из серии задачи, серия останется.", "\n➡️ This day leaves the task's series; the series stays."))
	default:
		b.WriteString(i18n.T(lang, "\n➡️ Задача исчезнет из списка, останется событие.", "\n➡️ The task leaves the list; the event remains."))
	}
	return b.String()
}

func (h *Handler) finishToEvent(chatID int64, messageID int, task api.Task) {
	us := h.store.GetOrCreate(chatID)
	start, err := time.Parse(time.RFC3339, fmt.Sprint(us.FlowData["start"]))
	minutes, _ := us.FlowData["minutes"].(int)
	mode, _ := us.FlowData["mode"].(string)
	repeat, _ := us.FlowData["repeat"].(string)
	if err != nil || minutes <= 0 || mode == "" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Это меню устарело — открой задачу заново.", "This menu is out of date — open the task again."),
			keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	end := start.Add(time.Duration(minutes) * time.Minute)
	ev, err := h.api.ConvertTask(us.AuthToken, task.ID, api.ConvertReq{
		Mode: mode, Repeat: repeat,
		StartsAt: start.UTC().Format(time.RFC3339), EndsAt: end.UTC().Format(time.RFC3339),
	})
	h.store.ClearFlow(chatID)
	if err != nil {
		reason := h.errorText(chatID, err)
		if api.CodeOf(err) == "NOT_AN_OCCURRENCE" {
			// Here it means a slot on a day the series skips — not «the series
			// ended», which is what the same code means under ✅.
			reason = h.t(chatID,
				"этот день не входит в серию. Выбери день по её расписанию.",
				"that day is not in the series. Pick a day it has.")
		}
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не получилось: ", "❌ Did not work: ")+format.Escape(reason),
			keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	now := time.Now().In(h.location(chatID))
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "✅ <b>В календаре</b>\n🕐 %s", "✅ <b>On the calendar</b>\n🕐 %s"),
			humanRange(h.lang(chatID), start.In(now.Location()), end.In(now.Location()), now)),
		keyboards.EventCard(h.lang(chatID), ev.ID))
}

// convertExplanation is the «ℹ️ Что это?» of both directions. It moves to the
// screen-explanation registry with piece C.
func convertExplanation(lang i18n.Lang) string {
	return i18n.T(lang,
		"ℹ️ <b>Задача и событие</b>\n\nЗадача — что сделать. Событие — когда.\n\n🔗 <b>Связать</b> — живут оба: событие даёт время и напоминания, задача закрывается только своим «Готово».\n➡️ <b>Перенести</b> — остаётся одно из двух.\n\nЧто потеряется, называется до подтверждения. ❌ Отмена ничего не меняет.",
		"ℹ️ <b>Tasks and events</b>\n\nA task is what to do. An event is when.\n\n🔗 <b>Link</b> — both live: the event carries the time and reminders; the task closes only with its own «Done».\n➡️ <b>Move</b> — one of the two remains.\n\nAnything lost is named before you confirm. ❌ Cancel changes nothing.")
}
