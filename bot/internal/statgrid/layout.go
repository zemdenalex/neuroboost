package statgrid

import "time"

// Period is what one screen of statistics covers.
type Period string

const (
	Week  Period = "week"
	Month Period = "month"
	Year  Period = "year"
	All   Period = "all"
)

// Scale is what a full bar means (spec §3): 24 hours by default, the working
// window, or the fullest cell on the screen.
type Scale struct {
	Kind      string
	WorkStart int
	WorkEnd   int
}

// Cell is one bar.
type Cell struct {
	From, To time.Time
	Out      bool
}

// Row is one line of the screen.
type Row struct {
	Label    int
	From, To time.Time
	Cells    []Cell
}

// Grid is one screen.
type Grid struct {
	Period   Period
	From, To time.Time
	ColHours []int
	Rows     []Row
}

func midnight(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

func monday(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

// Layout builds the cells of one period in the user's zone.
func Layout(p Period, now time.Time, offset int, loc *time.Location, sc Scale, firstYear int) Grid {
	today := midnight(now, loc)
	switch p {
	case Month:
		return layoutMonth(time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, offset, 0))
	case Year:
		return layoutYear(today.Year()+offset, loc)
	case All:
		return layoutAll(firstYear, today.Year(), loc)
	default:
		return layoutWeek(monday(today).AddDate(0, 0, 7*offset), sc)
	}
}

func layoutWeek(start time.Time, sc Scale) Grid {
	hours := make([]int, 0, 24)
	first, last := 0, 24
	if sc.Kind == "work" && sc.WorkEnd > sc.WorkStart {
		first, last = sc.WorkStart, sc.WorkEnd
	}
	for h := first; h < last; h++ {
		hours = append(hours, h)
	}
	g := Grid{Period: Week, From: start, To: start.AddDate(0, 0, 7), ColHours: hours}
	for d := 0; d < 7; d++ {
		day := start.AddDate(0, 0, d)
		row := Row{Label: d, From: day, To: day.AddDate(0, 0, 1)}
		for _, h := range hours {
			// Built from the date, not by adding hours: on a DST day the wall
			// clock and elapsed time disagree, and the wall clock is what the
			// column header says.
			from := time.Date(day.Year(), day.Month(), day.Day(), h, 0, 0, 0, day.Location())
			row.Cells = append(row.Cells, Cell{From: from, To: from.Add(time.Hour)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutMonth(first time.Time) Grid {
	next := first.AddDate(0, 1, 0)
	g := Grid{Period: Month, From: first, To: next}
	for wk := monday(first); wk.Before(next); wk = wk.AddDate(0, 0, 7) {
		_, isoWeek := wk.ISOWeek()
		row := Row{Label: isoWeek, From: wk, To: wk.AddDate(0, 0, 7)}
		for d := 0; d < 7; d++ {
			day := wk.AddDate(0, 0, d)
			row.Cells = append(row.Cells, Cell{From: day, To: day.AddDate(0, 0, 1),
				Out: day.Before(first) || !day.Before(next)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutYear(year int, loc *time.Location) Grid {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	g := Grid{Period: Year, From: start, To: start.AddDate(1, 0, 0)}
	for m := 0; m < 12; m++ {
		first := start.AddDate(0, m, 0)
		next := first.AddDate(0, 1, 0)
		row := Row{Label: m + 1, From: first, To: next}
		cellFrom := first
		for day := first.AddDate(0, 0, 1); day.Before(next); day = day.AddDate(0, 0, 1) {
			if day.Weekday() == time.Monday {
				row.Cells = append(row.Cells, Cell{From: cellFrom, To: day})
				cellFrom = day
			}
		}
		row.Cells = append(row.Cells, Cell{From: cellFrom, To: next})
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutAll(firstYear, lastYear int, loc *time.Location) Grid {
	if firstYear > lastYear {
		firstYear = lastYear
	}
	g := Grid{Period: All,
		From: time.Date(firstYear, 1, 1, 0, 0, 0, 0, loc),
		To:   time.Date(lastYear+1, 1, 1, 0, 0, 0, 0, loc)}
	for y := firstYear; y <= lastYear; y++ {
		start := time.Date(y, 1, 1, 0, 0, 0, 0, loc)
		row := Row{Label: y, From: start, To: start.AddDate(1, 0, 0)}
		for m := 0; m < 12; m++ {
			from := start.AddDate(0, m, 0)
			row.Cells = append(row.Cells, Cell{From: from, To: from.AddDate(0, 1, 0)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}
