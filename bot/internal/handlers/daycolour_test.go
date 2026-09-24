package handlers

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

// Spec §11: the future never, today only once taken, a day before the start
// only if the person chose so; after the start an untaken day is ⬛.
func TestDayColoursFollowTheRules(t *testing.T) {
	today := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	days := []api.Day{
		{Day: "2026-09-21", BeforeStart: true},
		{Day: "2026-09-22", Level: 0},
		{Day: "2026-09-23", Confirmed: true, Level: 3},
		{Day: "2026-09-24"},
		{Day: "2026-09-25", Confirmed: true, Level: 5},
	}
	got := dayColours(days, today, false)
	want := map[string]string{"2026-09-22": "⬛", "2026-09-23": "🟧"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not painting before: %v, want %v", got, want)
	}
	if got := dayColours(days, today, true); got["2026-09-21"] != "⬛" {
		t.Errorf("painting before: 21st = %q, want ⬛", got["2026-09-21"])
	}
	days[3].Confirmed, days[3].Level = true, 1
	if got := dayColours(days, today, false); got["2026-09-24"] != "🟫" {
		t.Errorf("today taken: %q, want 🟫", got["2026-09-24"])
	}
}

// Spec §6: colour, space, date, space, bar; the other month stays bare.
func TestTheCellCarriesColourDateAndBar(t *testing.T) {
	d := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	if got := cellLabel(dayCell{Date: d, InMonth: true, Level: 5, Colour: "🟩"}); got != "🟩 21 "+statgrid.Glyph(5) {
		t.Errorf("full cell = %q", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: true, Colour: "🟧"}); got != "🟧 21" {
		t.Errorf("no bar = %q", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: false, Colour: "🟧"}); got != "·21" {
		t.Errorf("other month = %q, want no colour", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: true, IsToday: true, Colour: "🟧"}); got != "🟧 🔸21" {
		t.Errorf("today = %q", got)
	}
}

// Through the handler: a taken today is coloured in the grid; switched off,
// no square anywhere.
func TestTheMonthShowsTheDayColour(t *testing.T) {
	today := moscowToday()
	taken := []map[string]any{{"day": today, "target": 5, "confirmed": true, "items": []any{}, "done": 3, "level": 3}}

	a := &dayAPI{days: taken}
	h, fake, chat := dayHandler(t, a)
	h.handleCalendar(chat, 0, time.Now())
	if got := fake.last(t).Markup; !strings.Contains(got, "🟧 🔸") {
		t.Errorf("a taken today is not coloured: %s", got)
	}

	a = &dayAPI{days: taken, settings: map[string]any{"day_tasks_enabled": false}}
	h, fake, chat = dayHandler(t, a)
	h.handleCalendar(chat, 0, time.Now())
	if got := fake.last(t).Markup; strings.Contains(got, "🟧") {
		t.Errorf("switched off, and a square is drawn: %s", got)
	}
}
