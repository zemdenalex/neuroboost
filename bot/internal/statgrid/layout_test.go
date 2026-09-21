package statgrid

import (
	"testing"
	"time"
)

var msk = mustLoc("Europe/Moscow")

func mustLoc(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

var day24 = Scale{Kind: "day24"}

// 22.09.2026 is a Tuesday.
var tue = time.Date(2026, 9, 22, 15, 0, 0, 0, msk)

func TestAWeekIsSevenDaysOfTwentyFourHours(t *testing.T) {
	g := Layout(Week, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 7 || len(g.Rows[0].Cells) != 24 || len(g.ColHours) != 24 {
		t.Fatalf("rows=%d cells=%d", len(g.Rows), len(g.Rows[0].Cells))
	}
	if !g.Rows[0].From.Equal(time.Date(2026, 9, 21, 0, 0, 0, 0, msk)) {
		t.Errorf("week starts %v, want Monday 21.09", g.Rows[0].From)
	}
	if c := g.Rows[1].Cells[10]; !c.From.Equal(time.Date(2026, 9, 22, 10, 0, 0, 0, msk)) || c.To.Sub(c.From) != time.Hour {
		t.Errorf("tue 10:00 cell = %v–%v", c.From, c.To)
	}
}

func TestWorkScaleShowsOnlyTheWorkingHours(t *testing.T) {
	g := Layout(Week, tue, 0, msk, Scale{Kind: "work", WorkStart: 8, WorkEnd: 20}, 2026)
	if len(g.ColHours) != 12 || g.ColHours[0] != 8 || len(g.Rows[0].Cells) != 12 {
		t.Errorf("cols = %v", g.ColHours)
	}
}

func TestOffsetsWalkBothWays(t *testing.T) {
	back := Layout(Week, tue, -3, msk, day24, 2026)
	if !back.From.Equal(time.Date(2026, 8, 31, 0, 0, 0, 0, msk)) {
		t.Errorf("-3 weeks from = %v", back.From)
	}
	next := Layout(Month, tue, 1, msk, day24, 2026)
	if next.From.Month() != time.October {
		t.Errorf("+1 month = %v", next.From)
	}
}

// September 2026 starts on a Tuesday and ends on a Wednesday: five ISO weeks,
// the first with one day outside (Mon 31.08), the last with four.
func TestAMonthIsItsWeeksWithTheOutsideDaysMarked(t *testing.T) {
	g := Layout(Month, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(g.Rows))
	}
	if !g.Rows[0].Cells[0].Out || g.Rows[0].Cells[1].Out {
		t.Errorf("first row: mon out=%v tue out=%v", g.Rows[0].Cells[0].Out, g.Rows[0].Cells[1].Out)
	}
	if !g.Rows[4].Cells[3].Out || g.Rows[4].Cells[2].Out {
		t.Errorf("last row: wed out=%v thu out=%v", g.Rows[4].Cells[2].Out, g.Rows[4].Cells[3].Out)
	}
}

// A year is twelve months; September 2026 cut at Mondays: 1–6, 7–13, 14–20,
// 21–27, 28–30 → five pieces.
func TestAYearIsMonthsCutAtMondays(t *testing.T) {
	g := Layout(Year, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 12 {
		t.Fatalf("rows = %d", len(g.Rows))
	}
	sep := g.Rows[8]
	if len(sep.Cells) != 5 || sep.Cells[1].From.Day() != 7 || sep.Cells[4].To.Day() != 1 {
		t.Errorf("september cells = %d, second from %d", len(sep.Cells), sep.Cells[1].From.Day())
	}
}

func TestAllIsYearsTimesMonths(t *testing.T) {
	g := Layout(All, tue, 5, msk, day24, 2025)
	if len(g.Rows) != 2 || g.Rows[0].Label != 2025 || len(g.Rows[1].Cells) != 12 {
		t.Errorf("rows = %d, first = %d", len(g.Rows), g.Rows[0].Label)
	}
}

// 🔴 The day the clocks go back is 25 hours long in New York. A layout built
// with +24h would put the next midnight at 23:00.
func TestDayBoundariesSurviveDST(t *testing.T) {
	ny := mustLoc("America/New_York")
	g := Layout(Week, time.Date(2026, 11, 1, 12, 0, 0, 0, ny), 0, ny, day24, 2026)
	sun := g.Rows[6]
	if sun.To.Sub(sun.From) != 25*time.Hour || sun.To.In(ny).Hour() != 0 {
		t.Errorf("sunday 01.11 = %v → %v", sun.From, sun.To)
	}
}
