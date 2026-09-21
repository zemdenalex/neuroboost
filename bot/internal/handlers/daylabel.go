package handlers

import (
	"fmt"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// dayLabel names a day in the reader's language.
//
// 🔴 Go's Format("Mon, Jan 2") is English whatever the chat's language, and on
// 21.09 Denis, with the interface in Russian, read:
//
//	🎯 Сегодня — Mon, Sep 21
//	📅 Due: Tue, Sep 22
//
// ⚠ Why no test caught it. TestNoUntranslatedUserFacingText looks for a
// CYRILLIC literal sitting outside an i18n.T call — untranslated Russian. An
// English-only literal has no Cyrillic in it, so the scan is blind to this by
// construction, and «Mon, Jan 2» is not even a literal a reader would notice:
// it is Go's reference date.
//
// Short weekday on purpose: this goes in a header next to other words, and
// «понедельник, 21 сентября» crowds it out.
func dayLabel(lang i18n.Lang, t time.Time) string {
	if lang != i18n.RU {
		return t.Format("Mon, Jan 2")
	}
	// ⚠ weekdayShort is ISO-indexed — Monday is 0 — while time.Weekday starts
	// at Sunday. The shift is the same one calendar.go uses to lay out a month
	// grid. Adding a second, Sunday-first table of day names instead would put
	// two conventions in one package, and the one that is wrong would be wrong
	// only on Sundays.
	return fmt.Sprintf("%s, %d %s",
		weekdayShort(lang, (int(t.Weekday())+6)%7), t.Day(), monthGenitive(lang, t.Month()))
}

// eventWhen says WHEN an event is, both ends of it.
//
// 🔴 Настя, 21.09, looking at «🕐 14:50 — Оркестр»: *«И почему он границу во
// времени не пишет? Я писала, со скольки до скольки»* — and she had. The end
// was in the data all along; the list simply never printed it, while the list
// of events one screen away did. Two renderings of the same thing, and only
// one of them complete.
func (h *Handler) eventWhen(chatID int64, e api.Event) string {
	tz := h.timezone(chatID)
	if e.AllDay {
		return h.t(chatID, "весь день", "all day")
	}
	start := format.FormatTime(e.StartsAt, tz)
	end := format.FormatTime(e.EndsAt, tz)
	if end == "" || end == start || e.EndsAt == "" {
		return start
	}
	return start + "–" + end
}

// tasksDueOn picks the tasks that belong to one calendar day.
//
// 🔴 Compared as DATES, never as instants. A due date has no time of day, so
// parsing it into a zone and comparing moments is how «22.09» lands on the
// 21st for everyone west of the server — the same mistake that made
// «каждый понедельник» surface on Tuesdays (recurrence.dayOf, 20.09).
//
// ⚠ A repeating task is listed only on its due date, not on every day of its
// series. Showing the series here needs the occurrence arithmetic the API
// holds, and guessing it in the bot would put a second opinion about what a
// repeat means into a second process.
func tasksDueOn(tasks []api.Task, day time.Time) []api.Task {
	want := day.Format("2006-01-02")
	out := []api.Task{}
	for _, t := range tasks {
		if t.DueDate == "" {
			continue
		}
		if len(t.DueDate) >= 10 && t.DueDate[:10] == want {
			out = append(out, t)
		}
	}
	return out
}

// dayLabelISO is dayLabel for a date that arrives as a string.
//
// A date-only value ("2026-09-22") is parsed as a DATE, not shifted into a
// zone: a due date has no time of day, and putting one on it is how «Due: 22»
// becomes «Due: 21» for everyone west of the server.
func dayLabelISO(lang i18n.Lang, iso string, tz string) string {
	if day, err := time.Parse("2006-01-02", iso); err == nil {
		return dayLabel(lang, day)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return dayLabel(lang, t.In(loc))
}
