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
	// h.location() rather than LoadLocation with a dropped error: that returned
	// a nil *Location on a bad TZ name, and time.Date panics on nil — in the
	// handler behind the most-pressed button in the bot.
	loc := h.location()
	now := time.Now().In(loc)

	// 🔴 Built in the user's zone, then converted. This used to stamp the local
	// calendar date with time.UTC, which for Moscow queried 03:00–03:00: three
	// hours of yesterday evening included, tonight's last three hours missing.
	// "Today" looked right most of the time, which is why it survived.
	from, to := dayBounds(now, loc)

	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID,
			"❌ Не удалось загрузить события: "+err.Error(), keyboards.BackToMenu())
		return
	}

	tasks, err := h.api.GetTasks(us.AuthToken, "TODO")
	if err != nil {
		tasks = nil
	}

	text := fmt.Sprintf("🎯 <b>Today's Focus</b> — %s\n🕐 %s (%s)\n\n",
		now.Format("Mon, Jan 2"),
		now.Format("15:04"),
		h.cfg.Timezone,
	)

	text += fmt.Sprintf("📅 <b>Events: %d</b>\n", len(events))
	sort.Slice(events, func(i, j int) bool { return events[i].StartsAt < events[j].StartsAt })
	for _, e := range events {
		text += fmt.Sprintf("  %s — %s\n", format.FormatTime(e.StartsAt, h.cfg.Timezone), format.Escape(e.Title))
	}

	if len(tasks) > 0 {
		text += fmt.Sprintf("\n🎯 <b>Tasks: %d</b>\n", len(tasks))
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
			text += fmt.Sprintf("  %s %s%s\n", format.PriorityEmoji(t.Priority), format.Escape(t.Title), dur)
		}
	}

	h.editOrSend(chatID, messageID, text, keyboards.BackToMenu())
}

func (h *Handler) handleStats(chatID int64, messageID int) {
	h.editOrSend(chatID, messageID,
		"📊 <b>Stats</b>\n\nComing soon! Track your productivity trends here.",
		keyboards.BackToMenu())
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
