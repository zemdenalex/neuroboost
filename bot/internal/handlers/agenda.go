package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// agendaHorizon is how far forward 📅 События looks.
//
// Two weeks rather than "everything": the screen answers "what is next", and a
// list long enough to scroll answers a different question — that one belongs to
// the month grid.
const agendaHorizon = 14 * 24 * time.Hour

// agendaText renders the upcoming list. Pure, so the day labels and the
// ordering are testable without a bot or an API.
func agendaText(events []api.Event, now time.Time, tz string) string {
	if len(events) == 0 {
		return "📅 <b>Ближайшие события</b>\n─────────────\nНичего не запланировано на две недели вперёд."
	}

	sorted := make([]api.Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartsAt < sorted[j].StartsAt })

	loc := now.Location()
	if l, err := time.LoadLocation(tz); err == nil {
		loc = l
	}
	today := now.In(loc).Format("2006-01-02")
	tomorrow := now.In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	text := "📅 <b>Ближайшие события</b>\n─────────────\n"
	lastDay := ""
	for _, e := range sorted {
		start, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil {
			// A row we cannot place in time is still a row the user owns. Show
			// it without a heading rather than dropping it silently.
			text += fmt.Sprintf("🕐 — %s\n", format.Escape(e.Title))
			continue
		}
		day := start.In(loc).Format("2006-01-02")
		if day != lastDay {
			switch day {
			case today:
				text += "\n<b>Сегодня</b>\n"
			case tomorrow:
				text += "\n<b>Завтра</b>\n"
			default:
				text += fmt.Sprintf("\n<b>%s</b>\n", start.In(loc).Format("02.01, Mon"))
			}
			lastDay = day
		}
		text += fmt.Sprintf("🕐 %s — %s\n",
			format.FormatTime(e.StartsAt, tz), format.Escape(e.Title))
	}
	return text
}

func (h *Handler) handleAgenda(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()
	now := time.Now().In(loc)
	from := now.UTC().Format(time.RFC3339)
	to := now.Add(agendaHorizon).UTC().Format(time.RFC3339)

	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.", keyboards.HomeInline())
		return
	}
	h.editOrSend(chatID, messageID, agendaText(events, now, h.cfg.Timezone), keyboards.AgendaActions())
}
