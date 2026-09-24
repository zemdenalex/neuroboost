package handlers

import (
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
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
