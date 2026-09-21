package statgrid

import (
	"testing"
	"time"
)

func TestLevelsAreEightStepsAndNothingIsADot(t *testing.T) {
	for _, c := range []struct {
		v, cap time.Duration
		want   int
	}{
		{0, time.Hour, 0},
		{time.Minute, time.Hour, 1}, // anything booked is visible
		{30 * time.Minute, time.Hour, 4},
		{time.Hour, time.Hour, 8},
		{3 * time.Hour, time.Hour, 8}, // over 100% → full
	} {
		if got := Level(c.v, c.cap); got != c.want {
			t.Errorf("Level(%v/%v) = %d, want %d", c.v, c.cap, got, c.want)
		}
	}
	if Glyph(0) != "·" || Glyph(8) != "█" || Glyph(1) != "▁" {
		t.Errorf("glyphs %q %q %q", Glyph(0), Glyph(1), Glyph(8))
	}
}

// Spec §7: 12 h booked in a day is half a bar at 24 h and a full one at 08–20.
func TestTheScaleDecidesTheHeight(t *testing.T) {
	g := Layout(Month, tue, 0, msk, day24, 2026)
	twelve := func(c Cell) time.Duration {
		if c.From.Day() == 22 && c.From.Month() == time.September {
			return 12 * time.Hour
		}
		return 0
	}
	find := func(lv [][]int) int {
		for r, row := range g.Rows {
			for i, c := range row.Cells {
				if c.From.Day() == 22 && c.From.Month() == time.September {
					return lv[r][i]
				}
			}
		}
		return -1
	}
	if l := find(Levels(g, day24, twelve)); l != 4 {
		t.Errorf("24h scale: level %d, want 4", l)
	}
	if l := find(Levels(g, Scale{Kind: "work", WorkStart: 8, WorkEnd: 20}, twelve)); l != 8 {
		t.Errorf("work scale: level %d, want 8", l)
	}
	if l := find(Levels(g, Scale{Kind: "peak"}, twelve)); l != 8 {
		t.Errorf("peak scale: the fullest cell is full, got %d", l)
	}
}
