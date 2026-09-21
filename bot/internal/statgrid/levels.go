package statgrid

import "time"

var glyphs = []string{"·", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// Glyph is the character for a level. Denis 20.09 on the old «▪»: «странный
// выбор черных квадратов» — height by time replaced it on 21.09.
func Glyph(level int) string {
	if level < 0 {
		level = 0
	}
	if level > 8 {
		level = 8
	}
	return glyphs[level]
}

// Level maps a share of capacity to 0..8. Anything booked is at least 1: a
// ten-minute call must not vanish from the screen.
func Level(value, capacity time.Duration) int {
	if value <= 0 || capacity <= 0 {
		return 0
	}
	l := int((value*8 + capacity - 1) / capacity) // ceil
	if l < 1 {
		l = 1
	}
	if l > 8 {
		l = 8
	}
	return l
}

// Capacity is a cell's time at full height under the scale.
func Capacity(c Cell, sc Scale) time.Duration {
	length := c.To.Sub(c.From)
	if length <= time.Hour || sc.Kind != "work" || sc.WorkEnd <= sc.WorkStart {
		return length
	}
	days := 0
	for d := c.From; d.Before(c.To); d = d.AddDate(0, 0, 1) {
		days++
	}
	return time.Duration(days*(sc.WorkEnd-sc.WorkStart)) * time.Hour
}

// Levels fills the grid. Cells outside the period stay 0.
func Levels(g Grid, sc Scale, value func(Cell) time.Duration) [][]int {
	vals := make([][]time.Duration, len(g.Rows))
	var peak time.Duration
	for r, row := range g.Rows {
		vals[r] = make([]time.Duration, len(row.Cells))
		for i, c := range row.Cells {
			if c.Out {
				continue
			}
			v := value(c)
			vals[r][i] = v
			if v > peak {
				peak = v
			}
		}
	}
	out := make([][]int, len(g.Rows))
	for r, row := range g.Rows {
		out[r] = make([]int, len(row.Cells))
		for i, c := range row.Cells {
			capacity := Capacity(c, sc)
			if sc.Kind == "peak" {
				capacity = peak
			}
			out[r][i] = Level(vals[r][i], capacity)
		}
	}
	return out
}
