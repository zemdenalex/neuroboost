package handlers

import (
	"fmt"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// dayColours is the square each day gets (spec 2026-09-22 §11): the future
// never, today only once taken, a day before the start only if the person
// chose so. After the start an untaken day is ⬛ — «не выбрал = не сделал».
// Keyed YYYY-MM-DD; a day with no square is absent.
func dayColours(days []api.Day, today time.Time, paintBefore bool) map[string]string {
	t := today.Format("2006-01-02")
	out := map[string]string{}
	for _, d := range days {
		switch {
		case d.Day > t:
		case d.Day == t && !d.Confirmed:
		case d.BeforeStart && !paintBefore:
		default:
			out[d.Day] = format.DayLevel(d.Level)
		}
	}
	return out
}

// todayDayLine is the day-tasks line under the Today title (spec §11,
// Denis 24.09: a line and a button, not the whole set).
func todayDayLine(lang i18n.Lang, d api.Day) string {
	if !d.Confirmed {
		return i18n.T(lang, "📌 День ещё не взят", "📌 The day is not taken yet")
	}
	return fmt.Sprintf(i18n.T(lang, "📌 %s %d из %d", "📌 %s %d of %d"), format.DayLevel(d.Level), d.Done, d.Target)
}
