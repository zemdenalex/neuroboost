package handlers

import (
	"strings"
	"testing"
	"time"
)

// Denis 24.09 (I2, spec §6): what a month cell shows is chosen in
// ⚙️ → 📌 Задачи дня: colour and bar, colour only, bar only. The date is
// always there: a cell without it cannot be picked.
func TestTheCellChoiceIsSavedAndTicked(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleDaySettings(990, 0, "cell_bar")
	if !strings.Contains(*patched, `"calendar_cell":"bar"`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "✓ ▅") {
		t.Errorf("bar only is not ticked: %s", m)
	}
	if got := h.calendarCell(990); got != cellBar {
		t.Errorf("cached cell = %q, want %q", got, cellBar)
	}
}

// Callback data is user input: an unknown choice writes nothing.
func TestAnUnknownCellWritesNothing(t *testing.T) {
	h, _, patched := scaleAPI(t)
	h.handleDaySettings(991, 0, "cell_x")
	if *patched != "" {
		t.Errorf("wrote %s", *patched)
	}
}

func TestApplyCell(t *testing.T) {
	levels := map[string]int{"2026-09-21": 3}
	colours := map[string]string{"2026-09-21": "🟩"}
	for _, c := range []struct {
		cell               string
		wantLevel, wantCol bool
	}{
		{cellBoth, true, true},
		{"", true, true},
		{cellColour, false, true},
		{cellBar, true, false},
	} {
		l, col := applyCell(c.cell, levels, colours)
		if (l != nil) != c.wantLevel || (col != nil) != c.wantCol {
			t.Errorf("%q: levels %v colours %v", c.cell, l, col)
		}
	}
}

// Bar only: the month does not read day tasks at all. The positive control
// (both) reads them, so the check can fail.
func TestABarOnlyMonthDoesNotReadDayTasks(t *testing.T) {
	for _, c := range []struct {
		cell string
		want bool
	}{{cellBar, false}, {cellBoth, true}} {
		a := &dayAPI{}
		h, _, chat := dayHandler(t, a)
		us := h.store.GetOrCreate(chat)
		us.CalendarCell, us.CalendarCellKnown = c.cell, true
		now := time.Now()
		h.showMonth(chat, 0, now.Year(), now.Month())
		read := false
		for _, call := range a.calls {
			if strings.HasPrefix(call, "GET /api/day-tasks") {
				read = true
			}
		}
		if read != c.want {
			t.Errorf("%s: day tasks read = %v, want %v (%v)", c.cell, read, c.want, a.calls)
		}
	}
}
