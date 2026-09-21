package handlers

import (
	"fmt"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// openTasks is what the lists show: waiting and scheduled alike.
//
// 🔴 SCHEDULED is set by the quick «⏰ Запланировать» (the web relies on it),
// and every screen here used to ask the API for TODO only — so scheduling a
// task made it vanish from the bot. Denis 21.09: it stays, marked with its time.
func openTasks(tasks []api.Task) []api.Task {
	out := make([]api.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == "TODO" || t.Status == "SCHEDULED" {
			out = append(out, t)
		}
	}
	return out
}

// linkedEvents finds, for every task, the event its time went into — the
// nearest one starting today or later.
//
// The API needs no new door for this: GET /api/events already carries
// task_id, and occurrences of a series are copies of their parent, so one
// fetch over a window answers every task on the screen.
func linkedEvents(events []api.Event, from time.Time) map[string]api.Event {
	dayStart := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	out := map[string]api.Event{}
	starts := map[string]time.Time{}
	for _, e := range events {
		if e.TaskID == nil || *e.TaskID == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil || at.Before(dayStart) {
			continue
		}
		if prev, ok := starts[*e.TaskID]; !ok || at.Before(prev) {
			starts[*e.TaskID] = at
			out[*e.TaskID] = e
		}
	}
	return out
}

// whenShort names a moment the way the list line has room for: «сегодня
// 15:00», «завтра 15:00», «пт 15:00» within the week, «22.10 15:00» further.
func whenShort(lang i18n.Lang, t, now time.Time) string {
	t = t.In(now.Location())
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location()) }
	hhmm := t.Format("15:04")
	switch days := int(day(t).Sub(day(now)).Hours() / 24); {
	case days == 0:
		return i18n.T(lang, "сегодня ", "today ") + hhmm
	case days == 1:
		return i18n.T(lang, "завтра ", "tomorrow ") + hhmm
	case days > 1 && days < 7:
		return weekdayShort(lang, (int(t.Weekday())+6)%7) + " " + hhmm
	default:
		return fmt.Sprintf("%02d.%02d %s", t.Day(), int(t.Month()), hhmm)
	}
}

// linkedWindowDays bounds the look-ahead; a link further out shows no mark.
const linkedWindowDays = 60

// linkedFor fetches today … +60 days and never fails.
func (h *Handler) linkedFor(chatID int64) map[string]api.Event {
	us := h.store.GetOrCreate(chatID)
	now := time.Now().In(h.location(chatID))
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	events, err := h.api.GetEvents(us.AuthToken,
		from.UTC().Format(time.RFC3339), from.AddDate(0, 0, linkedWindowDays).UTC().Format(time.RFC3339))
	if err != nil {
		// The mark is decoration; the list is the point. No events → no
		// marks, and the list still opens.
		return map[string]api.Event{}
	}
	return linkedEvents(events, now)
}
