package format

// dayLevels are the six colours of a day (spec 2026-09-22 §3, Denis's own
// table): 0 ⬛ nothing or not taken · 1 🟫 · 2 🟥 · 3 🟧 · 4 🟨 · 5 🟩 all done.
var dayLevels = [...]string{"⬛", "🟫", "🟥", "🟧", "🟨", "🟩"}

// DayLevel draws the level the server computed. The bot never computes it:
// two computations would one day paint one day twice.
func DayLevel(level int) string {
	if level < 0 || level >= len(dayLevels) {
		return dayLevels[0]
	}
	return dayLevels[level]
}
