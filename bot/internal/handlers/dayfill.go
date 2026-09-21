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
		if !e.AllDay {
			continue
		}
		// 🔴 Stored as INSTANTS at the user's local midnight (21:00Z for
		// Moscow, checked on dev 22.09), sometimes at an odd hour. The day is
		// where the instant falls in the user's zone — not the date printed in
		// the string, which for Moscow is the day before.
		s, err1 := time.Parse(time.RFC3339, e.StartsAt)
		en, err2 := time.Parse(time.RFC3339, e.EndsAt)
		if err1 != nil || err2 != nil {
			continue
		}
		start := midnightIn(s, loc)
		end := midnightIn(en, loc)
		if !en.In(loc).Equal(end) {
			end = end.AddDate(0, 0, 1) // an end inside a day includes that day
		}
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
		for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
			allDay[d.Format("2006-01-02")] = true
		}
	}

	return levelsByDay(spans, allDay, from, to, sc)
}

func midnightIn(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

func levelsByDay(spans []statgrid.Span, allDay map[string]bool, from, to time.Time, sc statgrid.Scale) map[string]int {
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
