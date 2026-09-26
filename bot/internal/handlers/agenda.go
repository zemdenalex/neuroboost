package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
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
func agendaText(lang i18n.Lang, events []api.Event, now time.Time, tz string) string {
	if len(events) == 0 {
		return i18n.T(lang,
			"📅 <b>Ближайшие события</b>\n─────────────\nНичего не запланировано на две недели вперёд.",
			"📅 <b>Upcoming</b>\n─────────────\nNothing scheduled for the next two weeks.")
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

	text := i18n.T(lang, "📅 <b>Ближайшие события</b>\n─────────────\n", "📅 <b>Upcoming</b>\n─────────────\n")
	lastDay := ""
	for _, e := range sorted {
		start, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil {
			// A row we cannot place in time is still a row the user owns. Show
			// it without a heading rather than dropping it silently.
			text += fmt.Sprintf("🕐 · %s\n", format.Escape(e.Title))
			continue
		}
		day := start.In(loc).Format("2006-01-02")
		if day != lastDay {
			switch day {
			case today:
				text += i18n.T(lang, "\n<b>Сегодня</b>\n", "\n<b>Today</b>\n")
			case tomorrow:
				text += i18n.T(lang, "\n<b>Завтра</b>\n", "\n<b>Tomorrow</b>\n")
			default:
				// Our weekday, not Go's English «Mon» (Denis, 23.09).
				local := start.In(loc)
				text += fmt.Sprintf("\n<b>%s %s</b>\n", weekdayShort(lang, (int(local.Weekday())+6)%7), local.Format("02.01"))
			}
			lastDay = day
		}
		// Start–end, or «весь день» (Denis, 23.09: «в списке событий нет конца»).
		// No em-dash between time and title: user-facing text rule.
		when := format.FormatTime(e.StartsAt, tz)
		if e.AllDay {
			when = i18n.T(lang, "весь день", "all day")
		} else if end, err := time.Parse(time.RFC3339, e.EndsAt); err == nil && end.After(start) {
			when += "–" + format.FormatTime(e.EndsAt, tz)
		}
		text += fmt.Sprintf("🕐 %s · %s\n", when, format.Escape(e.Title))
	}
	return text
}

func (h *Handler) handleAgenda(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location(chatID)
	now := time.Now().In(loc)
	from := now.UTC().Format(time.RFC3339)
	to := now.Add(agendaHorizon).UTC().Format(time.RFC3339)

	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "⚠️ Не дозвонился до сервера. Попробуй через минуту.", "⚠️ Could not reach the server. Try again in a minute."), h.home(chatID))
		return
	}
	h.editOrSend(chatID, messageID, agendaText(h.lang(chatID), events, now, h.timezone(chatID)), keyboards.AgendaActions(h.lang(chatID)))
}
