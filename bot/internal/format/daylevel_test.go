package format

import "testing"

// Spec 2026-09-22 §3: 0…5 = ⬛🟫🟥🟧🟨🟩. The server computes the level; this
// only draws it, and anything outside the range draws as «nothing done».
func TestDayLevel(t *testing.T) {
	want := []string{"⬛", "🟫", "🟥", "🟧", "🟨", "🟩"}
	for level, glyph := range want {
		if got := DayLevel(level); got != glyph {
			t.Errorf("DayLevel(%d) = %q, want %q", level, got, glyph)
		}
	}
	for _, bad := range []int{-1, 6, 99} {
		if got := DayLevel(bad); got != "⬛" {
			t.Errorf("DayLevel(%d) = %q, want ⬛", bad, got)
		}
	}
}
