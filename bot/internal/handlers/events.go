package handlers

import (
	"fmt"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// Creating an event from the bot.
//
// Until 13.08 the bot could capture a task but not an event, while 🗓 Calendar
// answered "coming in a future update" — a calendar-first product whose phone
// half could not touch the calendar.
//
// One line, parsed ("Ужин завтра 19:00"). When the line carries no time the
// bot ASKS with buttons instead of guessing: a wrong time written silently into
// a calendar is worse than one extra tap.
//
// Reminders are deliberately not sent: POST /api/events applies the user's
// default preset when reminder_offsets is absent (events/handlers.go), so an
// event created here is reachable by notifications without the bot knowing the
// user's settings. Sending an explicit empty array here would mean "silent
// forever" — the exact defect this product hit on 12.08.

func (h *Handler) startNewEventFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	h.sendHTML(chatID, "📅 <b>Новое событие</b>\n\n"+
		"Одной строкой, например:\n"+
		"<code>Ужин завтра 19:00</code>\n"+
		"<code>Созвон 14.08 15:00-16:30</code>\n\n"+
		"Можно просто название — спрошу время кнопками.")
}

// startNewEventForDay begins the normal new-event flow with the day already
// chosen, so "18:00 Ужин" is enough.
func (h *Handler) startNewEventForDay(chatID int64, date string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	us.FlowData["date"] = date
	h.sendHTML(chatID, "📅 <b>Событие на "+format.Escape(date)+"</b>\n\nНапиши время и название — например «18:00 Ужин».")
}

func (h *Handler) handleNewEventFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	if us.FlowStep != "line" {
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.HomeInline())
		return
	}

	if d, ok := us.FlowData["date"].(string); ok && d != "" {
		// parse.Parse understands "14.08"; feeding the chosen day in this form
		// reuses the tested path instead of adding a second way to set a date.
		if t, err := time.Parse("2006-01-02", d); err == nil {
			text = t.Format("02.01") + " " + text
		}
	}

	loc := h.location()
	res := parse.Parse(text, time.Now().In(loc))

	if res.Title == "" {
		h.sendText(chatID, "Нужно название события. Попробуй ещё раз или напиши cancel.")
		return
	}

	if res.NeedsTime {
		// Keep the title, ask only for what is missing.
		us.FlowData["title"] = res.Title
		us.FlowStep = "when"
		h.sendHTMLWithKeyboard(chatID,
			fmt.Sprintf("Когда — <b>%s</b>?", format.Escape(res.Title)),
			keyboards.EventWhen())
		return
	}

	// 0: the event came from typed text, so there is no screen of ours to edit.
	h.createEventAndReply(chatID, 0, res.Title, res.Start, res.End)
}

// handleWhenSelect turns a coarse button into a concrete time.
func (h *Handler) handleWhenSelect(chatID int64, messageID int, data string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_event" || us.FlowStep != "when" {
		return
	}
	title, _ := us.FlowData["title"].(string)
	if title == "" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, "Не помню название.", keyboards.RestartNewEvent())
		return
	}

	loc := h.location()
	now := time.Now().In(loc)
	var start time.Time

	switch data {
	case "when_now":
		// Rounded to the minute: a calendar entry starting at :37:412ms is noise.
		start = now.Truncate(time.Minute)
	case "when_hour":
		start = now.Add(time.Hour).Truncate(time.Minute)
	case "when_evening":
		start = time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, loc)
		// Asking for "this evening" after 19:00 means the next one.
		if !start.After(now) {
			start = start.AddDate(0, 0, 1)
		}
	case "when_tomorrow":
		start = time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1)
	default:
		return
	}

	h.createEventAndReply(chatID, messageID, title, start, start.Add(parse.DefaultDuration))
}

// createEventAndReply confirms a created event on the screen it was created
// from. messageID is 0 when the event came from typed text — there is no screen
// of ours to edit then, and the confirmation is posted fresh.
func (h *Handler) createEventAndReply(chatID int64, messageID int, title string, start, end time.Time) {
	us := h.store.GetOrCreate(chatID)

	ev, err := h.api.CreateEvent(us.AuthToken, api.CreateEventReq{
		Title:    title,
		StartsAt: start.UTC().Format(time.RFC3339),
		EndsAt:   end.UTC().Format(time.RFC3339),
	})
	h.store.ClearFlow(chatID)
	if err != nil {
		h.sendText(chatID, "❌ Не удалось создать: "+err.Error())
		return
	}

	loc := h.location()
	h.editOrSend(chatID, messageID, fmt.Sprintf("✅ <b>Событие создано</b>\n%s\n🕐 %s",
		format.Escape(ev.Title), humanRange(start.In(loc), end.In(loc), time.Now().In(loc))),
		keyboards.AgendaActions())
}

// humanRange renders "Завтра 19:00–20:00" — the same shape the reminder uses,
// so the confirmation and the later notification agree with each other.
func humanRange(start, end, now time.Time) string {
	day := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	hhmm := start.Format("15:04") + "–" + end.Format("15:04")

	switch days := int(day(start).Sub(day(now)).Hours() / 24); {
	case days == 0:
		return "Сегодня " + hhmm
	case days == 1:
		return "Завтра " + hhmm
	default:
		return fmt.Sprintf("%d %s %s", start.Day(), monthGenitive(start.Month()), hhmm)
	}
}

func monthGenitive(m time.Month) string {
	names := [...]string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	return names[int(m)-1]
}

// location resolves the configured timezone once, falling back to UTC rather
// than to whatever the container happens to be set to.
func (h *Handler) location() *time.Location {
	loc, err := time.LoadLocation(h.cfg.Timezone)
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}
