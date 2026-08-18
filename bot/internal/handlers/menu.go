package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// handleMenu draws the home screen — state first, buttons under it.
//
// 🔴 The summary must never be able to stop the screen from opening. This is a
// NAVIGATION screen: if the API is down, the user still needs the buttons that
// take them elsewhere. Both loads therefore degrade to a line of text rather
// than to an early return.
func (h *Handler) handleMenu(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()
	now := time.Now().In(loc)
	from, to := dayBounds(now, loc)

	text := fmt.Sprintf("🧠 <b>NeuroBoost</b> · %s\n─────────────\n", now.Format("02.01.2006"))

	events, evErr := h.api.GetEvents(us.AuthToken, from, to)
	tasks, tErr := h.api.GetTasks(us.AuthToken, "TODO")

	switch {
	case evErr != nil && tErr != nil:
		text += "Не смог загрузить сводку.\n"
	default:
		text += fmt.Sprintf("Сегодня: %d событий · %d задач\n", len(events), len(tasks))
		if len(events) > 0 {
			sort.Slice(events, func(i, j int) bool { return events[i].StartsAt < events[j].StartsAt })
			text += fmt.Sprintf("Ближайшее: %s %s\n",
				format.FormatTime(events[0].StartsAt, h.cfg.Timezone),
				format.Escape(events[0].Title))
		}
	}

	h.editOrSend(chatID, messageID, text, keyboards.HomeInline())
}
