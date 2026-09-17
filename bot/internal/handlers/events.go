package handlers

import (
	"fmt"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
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
func humanRange(lang i18n.Lang, start, end, now time.Time) string {
	day := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	hhmm := start.Format("15:04") + "–" + end.Format("15:04")

	switch days := int(day(start).Sub(day(now)).Hours() / 24); {
	case days == 0:
		return i18n.T(lang, "Сегодня ", "Today ") + hhmm
	case days == 1:
		return i18n.T(lang, "Завтра ", "Tomorrow ") + hhmm
	default:
		return fmt.Sprintf("%d %s %s", start.Day(), monthGenitive(lang, start.Month()), hhmm)
	}
}

// monthGenitive names a month the way a date reads it: «16 сентября», not
// «16 сентябрь». English has no such case, so the English column is the plain
// name and the function name stops being literally true there — kept anyway,
// because renaming it would break every caller for a word.
func monthGenitive(lang i18n.Lang, m time.Month) string {
	ru := [...]string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	en := [...]string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	return i18n.T(lang, ru[int(m)-1], en[int(m)-1])
}

// location is the user's own timezone, falling back to UTC rather than to
// whatever the container happens to be set to.
func (h *Handler) location(chatID int64) *time.Location {
	loc, err := time.LoadLocation(h.timezone(chatID))
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}

// timezone is this user's IANA zone, from their account.
//
// 🔴 It used to be h.cfg.Timezone — ONE zone for every user of the bot, set in
// the container's environment. Onboarding asks people what time it is for them
// (v0.4.11.2), and an answer that nothing reads would be a question that lies.
//
// Cached like the language, and like the language a failed read is NOT cached:
// it falls back to the configured zone for this one message and tries again
// on the next.
func (h *Handler) timezone(chatID int64) string {
	us := h.store.GetOrCreate(chatID)
	if us.TZKnown {
		return us.TZ
	}
	tz, err := h.api.MyTimezone(us.AuthToken)
	if err != nil || tz == "" {
		return h.cfg.Timezone
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return h.cfg.Timezone
	}
	us.TZ, us.TZKnown = tz, true
	return tz
}
