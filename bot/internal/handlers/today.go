package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleToday(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	loc, _ := time.LoadLocation(h.cfg.Timezone)
	now := time.Now().In(loc)

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.Add(24 * time.Hour)

	events, err := h.api.GetEvents(us.AuthToken, todayStart.Format(time.RFC3339), todayEnd.Format(time.RFC3339))
	if err != nil {
		h.sendText(chatID, "❌ Failed to load events: "+err.Error())
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

	h.sendHTMLWithKeyboard(chatID, text, keyboards.BackToMenu())
}

func (h *Handler) handleStats(chatID int64) {
	h.sendHTML(chatID, "📊 <b>Stats</b>\n\nComing soon! Track your productivity trends here.")
}
