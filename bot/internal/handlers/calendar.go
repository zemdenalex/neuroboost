package handlers

import "time"

// handleCalendar answers the 🗓 Calendar button.
//
// It used to be a dead end — "Calendar view coming in a future update" — which
// is what a user pressing it actually hit on 12.08 while looking for a way to
// add an event. A month view in a chat is still not the plan, but the button
// must at least name the two things that DO work, rather than apologising.
func (h *Handler) handleCalendar(chatID int64, date time.Time) {
	h.sendHTML(chatID, "🗓 <b>Календарь</b>\n\n"+
		"Создать событие — <b>📅 New Event</b>, одной строкой:\n"+
		"<code>Ужин завтра 19:00</code>\n\n"+
		"Что сегодня — <b>🎯 Today</b>.\n"+
		"Полный вид календаря — в вебе.")
}
