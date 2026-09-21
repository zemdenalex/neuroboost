package statgrid

import (
	"testing"
	"time"
)

func TestSeriesDaysFollowTheRule(t *testing.T) {
	from := time.Date(2026, 9, 21, 0, 0, 0, 0, msk)
	to := from.AddDate(0, 0, 7)
	anchor := "2026-09-01T00:00:00+03:00"
	for _, c := range []struct {
		rule string
		want int
	}{
		{"FREQ=DAILY", 7},
		{"FREQ=DAILY;INTERVAL=2", 4}, // anchored on the 1st: 21, 23, 25, 27
		{"FREQ=WEEKLY", 1},           // Tuesdays: 22.09
		{"FREQ=MONTHLY", 0},          // the 1st
		{"FREQ=DAILY;COUNT=25", 5},   // 1..25 → 21..25
		{"FREQ=DAILY;UNTIL=2026-09-23", 3},
		{"FREQ=YEARLY", 0}, // not written by the bot: unknown → 0
	} {
		if got := SeriesDays(c.rule, anchor, from, to, msk); got != c.want {
			t.Errorf("%s: %d, want %d", c.rule, got, c.want)
		}
	}
}

func TestASeriesThatStartsLaterHasNoEarlierDays(t *testing.T) {
	from := time.Date(2026, 9, 21, 0, 0, 0, 0, msk)
	if got := SeriesDays("FREQ=DAILY", "2026-09-25T00:00:00+03:00", from, from.AddDate(0, 0, 7), msk); got != 3 {
		t.Errorf("got %d, want 3 (25, 26, 27)", got)
	}
}
