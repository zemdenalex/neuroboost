package handlers

import "time"

func (h *Handler) handleCalendar(chatID int64, date time.Time) {
	h.sendHTML(chatID, "🗓 <b>Calendar</b>\n\nCalendar view coming in a future update.\nUse 🎯 Today to see today's events.")
}
