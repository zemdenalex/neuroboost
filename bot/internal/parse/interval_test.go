package parse

import "testing"

// Denis, 18.09: «и где свой вариант?» — the interval typed in answer.
func TestIntervalReadsTheWordsPeopleUse(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"40", 40},
		{"40 минут", 40},
		{"2 часа", 120},
		{"2ч", 120},
		{"3 hours", 180},
		{"90", 90},
		{"1 день", 1440},
	} {
		got, ok := Interval(c.in)
		if !ok || got != c.want {
			t.Errorf("Interval(%q) = %d, %v; want %d", c.in, got, ok, c.want)
		}
	}
}

// 🔴 Refused rather than clamped. The API caps a snooze at a day and would
// silently shorten anything longer — so a promise of three days would be kept
// as one, which is worse than being asked again.
func TestIntervalRefusesWhatTheAPIWouldSilentlyShorten(t *testing.T) {
	for _, bad := range []string{"", "потом", "0", "-5", "3 дня", "100 часов"} {
		if got, ok := Interval(bad); ok {
			t.Errorf("Interval(%q) = %d, accepted; it should be refused", bad, got)
		}
	}
}
