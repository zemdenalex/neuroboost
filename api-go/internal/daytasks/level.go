// Package daytasks is «задачи дня» (spec 2026-09-22): N tasks promised for a
// day, and the day's colour by how many of them were closed on it.
package daytasks

// Level is the colour of a day: 0 ⬛ · 1 🟫 · 2 🟥 · 3 🟧 · 4 🟨 · 5 🟩.
//
// 🔴 The server computes it and every client only draws it — the same rule as
// the fill of a day: two computations would one day paint one day twice.
//
// Denis, 22.09: N=3 counts what is left, every other N goes by share. At N=5
// both readings give his table exactly.
func Level(done, target int) int {
	if target <= 0 || done <= 0 {
		return 0
	}
	if done >= target {
		return 5
	}
	if target == 3 {
		return map[int]int{2: 4, 1: 3}[done]
	}
	share := done * 100 / target
	switch {
	case share >= 80:
		return 4
	case share >= 60:
		return 3
	case share >= 40:
		return 2
	default:
		return 1
	}
}
