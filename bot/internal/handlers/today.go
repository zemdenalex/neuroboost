package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleToday(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	// h.location(chatID) rather than LoadLocation with a dropped error: that returned
	// a nil *Location on a bad TZ name, and time.Date panics on nil — in the
	// handler behind the most-pressed button in the bot.
	loc := h.location(chatID)
	now := time.Now().In(loc)

	// 🔴 Built in the user's zone, then converted. This used to stamp the local
	// calendar date with time.UTC, which for Moscow queried 03:00–03:00: three
	// hours of yesterday evening included, tonight's last three hours missing.
	// "Today" looked right most of the time, which is why it survived.
	from, to := dayBounds(now, loc)

	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "❌ Не удалось загрузить события: ", "❌ Could not load events: ")+h.errorText(chatID, err), keyboards.BackToMenu(h.lang(chatID)))
		return
	}

	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		tasks = nil
	}
	tasks = openTasks(tasks)

	text := fmt.Sprintf(h.t(chatID,
		"🎯 <b>Сегодня</b>: %s\n🕐 %s (%s)\n\n",
		"🎯 <b>Today's focus</b>: %s\n🕐 %s (%s)\n\n"),
		dayLabel(h.lang(chatID), now),
		now.Format("15:04"),
		h.timezone(chatID),
	)

	// «шт» after the count, asked for by Настя on 21.09: *«визуально не сразу
	// понятно, что значат эти цифры рядом»*. English needs no such word.
	text += fmt.Sprintf(h.t(chatID, "📅 <b>События: %d шт</b>\n", "📅 <b>Events: %d</b>\n"), len(events))
	sort.Slice(events, func(i, j int) bool { return events[i].StartsAt < events[j].StartsAt })
	for _, e := range events {
		text += fmt.Sprintf("  %s · %s\n", h.eventWhen(chatID, e), format.Escape(e.Title))
	}

	if len(tasks) > 0 {
		text += fmt.Sprintf(h.t(chatID, "\n🎯 <b>Задачи: %d шт</b>\n", "\n🎯 <b>Tasks: %d</b>\n"), len(tasks))
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].Priority < tasks[j].Priority })
		limit := 5
		if len(tasks) < limit {
			limit = len(tasks)
		}
		for _, t := range tasks[:limit] {
			dur := ""
			if t.EstimatedMinutes > 0 {
				dur = " ~" + format.Duration(t.EstimatedMinutes)
			}
			text += fmt.Sprintf("  %s %s%s\n", h.prio(chatID, t.Priority), format.Escape(t.Title), dur)
		}
		// 🔴 The header counts every task and the list shows five. On 21.09
		// that screen said «Задачи: 6» above five lines, and neither Denis nor
		// Настя could have known the sixth existed. A number that disagrees
		// with the list under it is worse than no number: it is not a count of
		// anything the reader can see.
		if rest := len(tasks) - limit; rest > 0 {
			text += fmt.Sprintf(h.t(chatID,
				"  … и ещё %d, кнопкой ниже\n",
				"  … and %d more, button below\n"), rest)
		}
	}

	h.editOrSend(chatID, messageID, text, keyboards.TodayScreen(h.lang(chatID)))
}

// dayBounds is the half-open UTC range covering one local calendar day.
//
// Extracted so the conversion can be asserted: the whole defect was invisible
// inside a handler that needs a store, a bot and a live API to run at all, and
// "today" is right in 21 of every 24 hours even when it is wrong.
//
// Sent as UTC rather than as an offset-bearing RFC3339 string, matching every
// other call this bot makes — one wire format, one thing to be wrong about.
func dayBounds(now time.Time, loc *time.Location) (string, string) {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start.UTC().Format(time.RFC3339), start.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
}
