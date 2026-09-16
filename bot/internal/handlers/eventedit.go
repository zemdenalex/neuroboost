package handlers

import (
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Editing an event from the bot.
//
// Denis, 16.09: «нет возможности редактировать событие, задачи можно выбрать и
// редактировать, а события нет».
//
// 🔴 It reuses the creation card rather than growing a second editor. The card
// already knows how to show nine fields and how to change each one; the only
// difference is the verb at the end — POST for a draft with no id, PATCH for a
// draft that has one. Two editors would be two places to forget a field, and
// this repository has already paid that bill once (api/index.ts against
// api/tasks.ts).

// eventPickLabel is what one event looks like as a button.
func eventPickLabel(e api.Event, loc *time.Location) string {
	title := strings.TrimSpace(e.Title)
	if title == "" {
		title = "—"
	}
	// Telegram wraps long labels badly; a button nobody can read is no better
	// than no button.
	if len([]rune(title)) > 28 {
		title = string([]rune(title)[:27]) + "…"
	}
	if t, err := time.Parse(time.RFC3339, e.StartsAt); err == nil {
		return t.In(loc).Format("02.01 15:04") + " · " + title
	}
	return title
}

// draftFromEvent loads an existing event into the same shape the creation card
// edits.
//
// ⚠ The times come back as instants and the draft holds a day plus offsets, so
// the conversion goes through the user's own zone — the same zone the card will
// print them in. Doing it in UTC would shift every event by the offset the
// moment it was opened, which reads as the bot corrupting data.
func draftFromEvent(e api.Event, loc *time.Location) draftState {
	st := draftState{Title: e.Title, EventID: e.ID, CalendarID: e.CalendarID}

	start, err := time.Parse(time.RFC3339, e.StartsAt)
	if err != nil {
		return st
	}
	local := start.In(loc)
	st.D.Day = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	st.D.HasDay = true

	if e.AllDay {
		st.D.AllDay = true
	} else {
		st.D.Start = local.Sub(st.D.Day)
		st.D.HasTime = true
		if end, err := time.Parse(time.RFC3339, e.EndsAt); err == nil {
			st.D.End = end.In(loc).Sub(st.D.Day)
			st.D.HasEnd = true
		}
	}

	if e.Rrule != nil && *e.Rrule != "" {
		st.D.Repeat = *e.Rrule
	}
	if e.Color != "" {
		st.D.Colour = e.Color
	}
	st.D.Tags = append([]string(nil), e.Tags...)
	if e.ReminderOffsets != nil {
		offsets := append([]int(nil), e.ReminderOffsets...)
		st.ReminderOffsets = &offsets
	}
	return st
}

// handleEventPicker turns the agenda into something tappable.
func (h *Handler) handleEventPicker(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()
	now := time.Now().In(loc)

	events, err := h.api.GetEvents(us.AuthToken,
		now.Format(time.RFC3339), now.Add(agendaHorizon).Format(time.RFC3339))
	if err != nil {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.",
			"⚠️ Could not reach the server. Try again in a minute."),
			keyboards.HomeInline(h.lang(chatID)))
		return
	}
	if len(events) == 0 {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"📅 Ближайших событий нет — редактировать нечего.",
			"📅 Nothing upcoming — there is nothing to edit."),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}

	sortEventsByStart(events)
	labels := make([]string, 0, len(events))
	ids := make([]string, 0, len(events))
	for _, e := range events {
		labels = append(labels, eventPickLabel(e, loc))
		ids = append(ids, e.ID)
	}

	text := h.t(chatID, "📅 <b>Какое событие открыть?</b>", "📅 <b>Which event?</b>")
	if len(events) > keyboards.EventPickLimit {
		// 🔴 Said out loud rather than silently truncated. A list that quietly
		// stops at twelve teaches that the thirteenth event does not exist.
		text += h.t(chatID,
			"\n\n<i>Показаны первые 12 из ", "\n\n<i>Showing the first 12 of ") +
			format.Escape(itoa(len(events))) +
			h.t(chatID, " — открой день в календаре, чтобы дойти до остальных.</i>",
				" — open a day in the calendar to reach the rest.</i>")
	}
	h.editOrSend(chatID, messageID, text, keyboards.EventPicker(h.lang(chatID), labels, ids, "agenda_open"))
}

// handleEventCard shows one event and what can be done to it.
func (h *Handler) handleEventCard(chatID int64, messageID int, eventID string) {
	us := h.store.GetOrCreate(chatID)
	parentID, _, isInstance := splitInstanceID(eventID)
	ev, err := h.api.GetEvent(us.AuthToken, parentID)
	if err != nil || ev == nil || ev.ID == "" {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"❌ Не удалось открыть событие — возможно, оно уже удалено.",
			"❌ Could not open the event — it may already be deleted."),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}

	h.store.ClearFlow(chatID)
	st := draftFromEvent(*ev, h.location())
	st.CalendarName = h.calendarName(chatID, ev.CalendarID)

	text := renderDraft(h.lang(chatID), st, time.Now().In(h.location()))
	if isInstance {
		// 🔴 Said BEFORE anything changes. The card shows the series — its own
		// first occurrence, not the one that was tapped — and a screen that
		// silently swapped the date under the user would be worse than one that
		// explains itself.
		text += h.t(chatID,
			"\n\n⚠ Это повторяющееся событие. Открыта вся серия, и изменения применятся ко всем повторам.",
			"\n\n⚠ This event repeats. The whole series is open, and changes apply to every occurrence.")
	}
	h.editOrSend(chatID, messageID, text, keyboards.EventCard(h.lang(chatID), parentID))
}

// handleEventEdit opens the existing event in the creation card.
func (h *Handler) handleEventEdit(chatID int64, messageID int, eventID string) {
	us := h.store.GetOrCreate(chatID)
	parentID, _, _ := splitInstanceID(eventID)
	ev, err := h.api.GetEvent(us.AuthToken, parentID)
	if err != nil || ev == nil || ev.ID == "" {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"❌ Не удалось открыть событие.", "❌ Could not open the event."),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}

	st := draftFromEvent(*ev, h.location())
	st.CalendarName = h.calendarName(chatID, ev.CalendarID)

	us.CurrentFlow = "new_event"
	us.FlowData = map[string]any{"draft": &st}
	h.showCard(chatID, messageID)
}

func (h *Handler) handleEventDeleteAsk(chatID int64, messageID int, eventID string) {
	h.editOrSend(chatID, messageID, h.t(chatID,
		"🗑 Удалить событие? Это нельзя отменить.\n\n⚠ У повторяющегося события удалится вся серия.",
		"🗑 Delete this event? This cannot be undone.\n\n⚠ A repeating event loses the whole series."),
		keyboards.EventDeleteConfirm(h.lang(chatID), eventID))
}

func (h *Handler) handleEventDelete(chatID int64, messageID int, eventID string) {
	us := h.store.GetOrCreate(chatID)
	parentID, _, _ := splitInstanceID(eventID)
	if err := h.api.DeleteEvent(us.AuthToken, parentID); err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не удалось удалить: ", "❌ Could not delete: ")+format.Escape(err.Error()),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}
	h.editOrSend(chatID, messageID,
		h.t(chatID, "🗑 Событие удалено.", "🗑 Event deleted."),
		keyboards.AgendaActions(h.lang(chatID)))
}

// updateFromDraft writes the edited card back to an event that already exists.
//
// 🔴 Every field is sent, not only the ones that look changed. The card is the
// whole truth about the event — the user may have CLEARED a colour or a repeat,
// and "send only what differs" cannot express a clearing without comparing
// against the original, which the draft no longer holds.
//
// ⚠ Tags are the exception the API forces: its Tags field is a plain slice with
// omitempty, so an empty list cannot be sent at all and tags can be added or
// replaced but not emptied from here. Named rather than hidden.
func (h *Handler) updateFromDraft(chatID int64, messageID int, st draftState) {
	us := h.store.GetOrCreate(chatID)
	start, end := draftBounds(st)

	title := st.Title
	startsAt := start.UTC().Format(time.RFC3339)
	endsAt := end.UTC().Format(time.RFC3339)
	allDay := st.D.AllDay
	rrule := st.D.Repeat
	colour := st.D.Colour

	req := api.UpdateEventReq{
		Title:           &title,
		StartsAt:        &startsAt,
		EndsAt:          &endsAt,
		AllDay:          &allDay,
		Rrule:           &rrule,
		Colour:          &colour,
		Tags:            st.D.Tags,
		ReminderOffsets: st.ReminderOffsets,
	}
	if st.CalendarID != "" {
		id := st.CalendarID
		req.CalendarID = &id
	}

	err := h.api.UpdateEvent(us.AuthToken, st.EventID, req)
	h.store.ClearFlow(chatID)
	if err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+format.Escape(err.Error()),
			keyboards.AgendaActions(h.lang(chatID)))
		return
	}

	loc := h.location()
	h.editOrSend(chatID, messageID,
		h.t(chatID, "💾 <b>Сохранено</b>\n", "💾 <b>Saved</b>\n")+
			format.Escape(st.Title)+"\n🕐 "+
			humanRange(h.lang(chatID), start.In(loc), end.In(loc), time.Now().In(loc)),
		keyboards.EventCard(h.lang(chatID), st.EventID))
}

// handleEventCallback answers the event screens. Returns false for anything it
// does not own.
//
// ⚠ The prefixes are checked longest-first: "evdy_" starts with "evd_", which
// starts with "ev_". Checking them the other way round would route every delete
// confirmation to the card, which is the kind of bug that looks like the button
// doing nothing.
func (h *Handler) handleEventCallback(chatID int64, messageID int, data string) bool {
	kind, id, ok := eventRoute(data)
	if !ok {
		return false
	}
	switch kind {
	case eventPick:
		h.handleEventPicker(chatID, messageID)
	case eventDelete:
		h.handleEventDelete(chatID, messageID, id)
	case eventDeleteAsk:
		h.handleEventDeleteAsk(chatID, messageID, id)
	case eventEdit:
		h.handleEventEdit(chatID, messageID, id)
	case eventOpen:
		h.handleEventCard(chatID, messageID, id)
	}
	return true
}

// What an event callback means.
const (
	eventPick      = "pick"
	eventOpen      = "open"
	eventEdit      = "edit"
	eventDeleteAsk = "delete-ask"
	eventDelete    = "delete"
)

// eventRoute decides what one callback means.
//
// ⚠ The four prefixes look nested and are not: the trailing UNDERSCORE is what
// separates them. "evdy_x" does not start with "evd_" — the fourth character
// is "y", not "_" — so the order of these four lines carries no meaning, and
// reversing it changes nothing. I wrote the opposite here first and a sabotage
// run disproved it; the claim is corrected rather than deleted, because the
// underscore is load-bearing and the next person to shorten a prefix needs to
// know why it is there.
//
// 🔴 What the test DOES hold is that each prefix routes where it says and that
// a prefix with no id is refused. Drop the underscore from any of them and the
// collision becomes real.
func eventRoute(data string) (kind, id string, ok bool) {
	if data == "event_pick" {
		return eventPick, "", true
	}
	for _, p := range []struct{ prefix, kind string }{
		{"evdy_", eventDelete},
		{"evd_", eventDeleteAsk},
		{"eve_", eventEdit},
		{"ev_", eventOpen},
	} {
		if strings.HasPrefix(data, p.prefix) {
			id = strings.TrimPrefix(data, p.prefix)
			if id == "" {
				// A prefix with no id addresses nothing. Routing it would call a
				// handler that fetches "" and reports the event missing, which
				// reads as data loss rather than a malformed button.
				return "", "", false
			}
			return p.kind, id, true
		}
	}
	return "", "", false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// sortEventsByStart orders a copy in place. The agenda already sorts for
// display; the picker needs the same order so the buttons match the list above
// them.
func sortEventsByStart(events []api.Event) {
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j].StartsAt < events[j-1].StartsAt; j-- {
			events[j], events[j-1] = events[j-1], events[j]
		}
	}
}
