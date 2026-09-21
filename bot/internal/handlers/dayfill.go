package handlers

import (
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

// dayLevels is how full each day in [from, to) is — 0 (absent) to 8.
//
// Denis 21.09 on the month calendar's dot: «по-хорошему везде они должны
// быть». The same statgrid measure and the same scale as statistics (spec
// §4): two screens that measured one day differently would disagree about
// the same Tuesday, and the user would have to decide which one lies.
func dayLevels(events []api.Event, from, to time.Time, sc statgrid.Scale, loc *time.Location) map[string]int {
	spans, _ := statgrid.EventSpans(events)

	// All-day events book no hours (spec §1), but their days are not empty.
	allDay := map[string]bool{}
	for _, e := range events {
		if !e.AllDay || len(e.StartsAt) < 10 || len(e.EndsAt) < 10 {
			continue
		}
		// An all-day event is dates, not instants: read the date part as is.
		start, err1 := time.ParseInLocation("2006-01-02", e.StartsAt[:10], loc)
		end, err2 := time.ParseInLocation("2006-01-02", e.EndsAt[:10], loc)
		if err1 != nil || err2 != nil {
			continue
		}
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
		for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
			allDay[d.Format("2006-01-02")] = true
		}
	}

	type day struct {
		cell statgrid.Cell
		busy time.Duration
	}
	var days []day
	var peak time.Duration
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		c := statgrid.Cell{From: d, To: d.AddDate(0, 0, 1)}
		b := statgrid.Busy(spans, c.From, c.To)
		days = append(days, day{cell: c, busy: b})
		if b > peak {
			peak = b
		}
	}

	out := map[string]int{}
	for _, d := range days {
		capacity := statgrid.Capacity(d.cell, sc)
		if sc.Kind == scalePeak {
			capacity = peak
		}
		key := d.cell.From.Format("2006-01-02")
		l := statgrid.Level(d.busy, capacity)
		if l == 0 && allDay[key] {
			l = 1
		}
		if l > 0 {
			out[key] = l
		}
	}
	return out
}
