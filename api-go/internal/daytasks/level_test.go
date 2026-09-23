package daytasks

import "testing"

// Spec §3. N=5 is Denis's table word for word; N=3 counts what is left; every
// other N goes by share. The table is the spec — a change here is a change to
// what the colours mean.
func TestLevel(t *testing.T) {
	cases := []struct{ done, target, want int }{
		// N=5: 🟩🟨🟧🟥🟫⬛
		{5, 5, 5}, {4, 5, 4}, {3, 5, 3}, {2, 5, 2}, {1, 5, 1}, {0, 5, 0},
		// N=3, by what is left: 3 🟩 · 2 🟨 · 1 🟧 · 0 ⬛
		{3, 3, 5}, {2, 3, 4}, {1, 3, 3}, {0, 3, 0},
		// N=4 by share: 100 · 75 · 50 · 25 · 0
		{4, 4, 5}, {3, 4, 3}, {2, 4, 2}, {1, 4, 1}, {0, 4, 0},
		// N=6: 100 · 83 · 66 · 50 · 33 · 16 · 0
		{6, 6, 5}, {5, 6, 4}, {4, 6, 3}, {3, 6, 2}, {2, 6, 1}, {1, 6, 1}, {0, 6, 0},
		// N=7: 100 · 85 · 71 · 57 · 42 · 28 · 14 · 0
		{7, 7, 5}, {6, 7, 4}, {5, 7, 3}, {4, 7, 2}, {3, 7, 2}, {2, 7, 1}, {1, 7, 1}, {0, 7, 0},
		// more done than promised is still a full day
		{6, 5, 5},
	}
	for _, c := range cases {
		if got := Level(c.done, c.target); got != c.want {
			t.Errorf("Level(%d, %d) = %d, want %d", c.done, c.target, got, c.want)
		}
	}
}

// A target outside 3–7 never reaches here (Target clamps it), but a zero would
// divide by zero — the function answers 0 rather than panicking the API.
func TestLevelWithNoTargetIsZero(t *testing.T) {
	if got := Level(3, 0); got != 0 {
		t.Errorf("Level(3, 0) = %d, want 0", got)
	}
}
