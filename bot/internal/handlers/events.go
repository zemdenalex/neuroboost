package handlers

import (
	"fmt"
	"time"
)

// What is left here after 15.09 are the shared helpers.
//
// The creation flow itself moved to draftflow.go when the confirmation card
// became mandatory. The old flow had two exits — ask-with-buttons, or create
// immediately — and Denis asked for one: parse, show what was understood, then
// confirm. Keeping a second path alive "just in case" would mean two places
// that build an event and two places to forget a field in.
//
// Gone with it: EventWhen and its four when_* callbacks. A keyboard whose
// handler no longer exists is a button that does nothing, and this repository
// has a rule about that — кнопка это заявка, обработчик это свидетельство.

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
