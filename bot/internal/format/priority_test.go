package format

import "testing"

// Настя, 18.09: «смайлики слишком из разных цветов, нет одного стиля».
// Denis 21.09: three styles to choose from, circles by default.
func TestPriorityHasThreeStyles(t *testing.T) {
	for _, c := range []struct {
		style string
		p     int
		want  string
	}{
		{StyleCircles, 1, "🔴"},
		{"", 1, "🔴"},        // no choice yet = circles
		{"weird", 3, "🟡"},   // an unknown value is not a blank
		{StyleDot, 1, "●1"}, // filled = urgent (1–2)
		{StyleDot, 2, "●2"},
		{StyleDot, 3, "○3"},
		{StyleDot, 0, "·0"}, // 0 = buffer
		{StyleDash, 1, "—"},
		{StyleDash, 5, "—"},
	} {
		if got := Priority(c.style, c.p); got != c.want {
			t.Errorf("Priority(%q, %d) = %q, want %q", c.style, c.p, got, c.want)
		}
	}
}
