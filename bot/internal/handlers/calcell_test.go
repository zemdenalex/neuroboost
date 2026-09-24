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
		l, col := applyCell(c.cell, true, levels, colours)
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

// Advisor 24.09: colour only, then day tasks switched off: the choice row is
// hidden, so a bare calendar would have no way back. Off draws the bar.
func TestColourOnlyWithDayTasksOffKeepsTheBar(t *testing.T) {
	levels := map[string]int{"2026-09-21": 3}
	if l, _ := applyCell(cellColour, false, levels, nil); l == nil {
		t.Error("day tasks off + colour only dropped the bar")
	}
}

// The same through showMonth: colour only, day tasks off, a busy day: the
// month still draws its bar.
func TestAColourOnlyMonthWithDayTasksOffDrawsBars(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	now := time.Now().In(loc)
	// Tomorrow: today's cell draws 🔸 instead of a bar, and tomorrow is
	// always inside the 42-day grid.
	start := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, loc)
	a := &dayAPI{settings: map[string]any{"day_tasks_enabled": false},
		events: []map[string]any{{"id": "e1", "title": "работа",
			"starts_at": start.UTC().Format(time.RFC3339), "ends_at": start.Add(8 * time.Hour).UTC().Format(time.RFC3339)}}}
	h, fake, chat := dayHandler(t, a)
	us := h.store.GetOrCreate(chat)
	us.CalendarCell, us.CalendarCellKnown = cellColour, true
	h.showMonth(chat, 0, now.Year(), now.Month())
	m := fake.last(t).Markup
	if !strings.ContainsAny(m, "▁▂▃▄▅▆▇█") {
		t.Errorf("no bar in the month: %s", m)
	}
}

// Final review M4: the month legend names the squares when they can appear,
// and says nothing about them when day tasks are off or the cell is bar only.
func TestTheMonthLegendNamesTheSquares(t *testing.T) {
	for _, c := range []struct {
		on   bool
		cell string
		want bool
	}{{true, cellBoth, true}, {true, cellBar, false}, {false, cellBoth, false}} {
		a := &dayAPI{settings: map[string]any{"day_tasks_enabled": c.on}}
		h, fake, chat := dayHandler(t, a)
		us := h.store.GetOrCreate(chat)
		us.CalendarCell, us.CalendarCellKnown = c.cell, true
		now := time.Now()
		h.showMonth(chat, 0, now.Year(), now.Month())
		if got := strings.Contains(fake.last(t).Text, "🟩"); got != c.want {
			t.Errorf("on=%v cell=%s: legend has squares = %v, want %v: %q", c.on, c.cell, got, c.want, fake.last(t).Text)
		}
	}
}
