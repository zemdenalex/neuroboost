package daytasks

import (
	"testing"
	"time"
)

// Spec §5, in the USER's zone: 11:59 in New York may remove today's task,
// 12:00 may not — whatever the server's clock says.
func TestCanRemoveIsNoonWhereTheUserIs(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) // a date; zone irrelevant
	at := func(h, m int) time.Time { return time.Date(2026, 9, 22, h, m, 0, 0, ny) }

	cases := []struct {
		name string
		now  time.Time
		day  time.Time
		want bool
	}{
		{"today 11:59", at(11, 59), day, true},
		{"today 12:00", at(12, 0), day, false},
		{"tomorrow, late", at(23, 0), day.AddDate(0, 0, 1), true},
		{"yesterday, early", at(8, 0), day.AddDate(0, 0, -1), false},
		// 03:00 UTC on the 23rd is still the 22nd, 23:00, in New York: it is
		// «today after noon» there, not «tomorrow morning».
		{"server already on the 23rd", time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC), day, false},
		{"server already on the 23rd, the 23rd is tomorrow", time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC), day.AddDate(0, 0, 1), true},
	}
	for _, c := range cases {
		if got := CanRemove(c.day, c.now, "America/New_York"); got != c.want {
			t.Errorf("%s: CanRemove = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTargetIsThreeToSevenElseFive(t *testing.T) {
	cases := map[string]int{
		`{}`:                       5,
		`{"day_tasks_target":3}`:   3,
		`{"day_tasks_target":7}`:   7,
		`{"day_tasks_target":2}`:   5,
		`{"day_tasks_target":8}`:   5,
		`{"day_tasks_target":"4"}`: 5, // a string is not a number the bot wrote
		`not json`:                 5,
	}
	for raw, want := range cases {
		if got := Target([]byte(raw)); got != want {
			t.Errorf("Target(%s) = %d, want %d", raw, got, want)
		}
	}
}
