package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func TestBuildMonthAlwaysStartsOnMondayAndFillsSixWeeks(t *testing.T) {
	loc := mustLoad(t, "Europe/Moscow")
	// August 2026 starts on a Saturday — the awkward case, where the first row
	// is almost entirely July. A month starting on Monday would hide an
	// off-by-one in the ISO offset entirely.
	cells := buildMonth(2026, time.August, time.Date(2026, 8, 18, 12, 0, 0, 0, loc), nil, loc)

	if len(cells) != 42 {
		t.Fatalf("got %d cells, want 42 — the grid must not change height between months", len(cells))
	}
	if got := cells[0].Date.Weekday(); got != time.Monday {
		t.Errorf("the grid starts on %s, want Monday", got)
	}
	if got := cells[0].Date.Format("2006-01-02"); got != "2026-07-27" {
		t.Errorf("first cell is %s, want 2026-07-27", got)
	}
	if cells[0].InMonth {
		t.Error("27 July is marked as inside August")
	}
	// 1 August is the sixth cell (Mon 27 … Sat 1).
	if got := cells[5].Date.Format("2006-01-02"); got != "2026-08-01" || !cells[5].InMonth {
		t.Errorf("cell 5 is %s (inMonth=%v), want 2026-08-01 inside the month", got, cells[5].InMonth)
	}
	// Consecutive days, no gaps or repeats.
	for i := 1; i < len(cells); i++ {
		if got := cells[i].Date.Sub(cells[i-1].Date); got != 24*time.Hour {
			t.Fatalf("cell %d is %v after its predecessor, want 24h", i, got)
		}
	}
}

func TestBuildMonthHandlesAMonthStartingOnMonday(t *testing.T) {
	// June 2026 starts on a Monday: the offset is zero and the grid must NOT
	// begin a week earlier. This is the case an `offset == 0 ? 7 : offset` slip
	// would break, and the Saturday fixture above cannot see it.
	loc := mustLoad(t, "Europe/Moscow")
	cells := buildMonth(2026, time.June, time.Date(2026, 6, 15, 12, 0, 0, 0, loc), nil, loc)
	if got := cells[0].Date.Format("2006-01-02"); got != "2026-06-01" {
		t.Errorf("first cell is %s, want 2026-06-01", got)
	}
	if !cells[0].InMonth {
		t.Error("1 June is not marked as inside June")
	}
}

func TestBuildMonthMarksTodayAndBusyDays(t *testing.T) {
	loc := mustLoad(t, "Europe/Moscow")
	today := time.Date(2026, 8, 18, 23, 40, 0, 0, loc)
	busy := map[string]bool{"2026-08-20": true, "2026-07-28": true}

	cells := buildMonth(2026, time.August, today, busy, loc)

	var sawToday, sawBusy, sawBusyOutside int
	for _, c := range cells {
		switch c.Date.Format("2006-01-02") {
		case "2026-08-18":
			if !c.IsToday {
				t.Error("18 August is not marked as today")
			}
			sawToday++
		case "2026-08-20":
			if !c.HasEvents {
				t.Error("20 August is not marked busy")
			}
			sawBusy++
		case "2026-07-28":
			// A busy day in the leading week: the grid queries the whole visible
			// range for exactly this reason.
			if !c.HasEvents {
				t.Error("28 July is not marked busy though it is on screen")
			}
			sawBusyOutside++
		}
	}
	if sawToday != 1 || sawBusy != 1 || sawBusyOutside != 1 {
		t.Fatalf("the fixture dates appear %d/%d/%d times — the assertions above may not have run",
			sawToday, sawBusy, sawBusyOutside)
	}
}

// The zones only diverge in the small hours: Moscow is UTC+3, so 01:30 on the
// 19th is 22:30 on the 18th in UTC. A grid comparing UTC dates marks yesterday.
//
// 🔴 The first version of this test used 23:40, where UTC and Moscow agree on
// the date — it asserted against a fixture that could not tell the two apart,
// and stayed green when the comparison was switched to UTC.
func TestTodayIsMarkedInTheUsersZoneNotUTC(t *testing.T) {
	loc := mustLoad(t, "Europe/Moscow")
	smallHours := time.Date(2026, 8, 19, 1, 30, 0, 0, loc)
	if smallHours.UTC().Day() == smallHours.Day() {
		t.Fatalf("fixture proves nothing: UTC and %s agree it is the %dth", loc, smallHours.Day())
	}

	cells := buildMonth(2026, time.August, smallHours, nil, loc)
	var marked []string
	for _, c := range cells {
		if c.IsToday {
			marked = append(marked, c.Date.Format("2006-01-02"))
		}
	}
	if len(marked) != 1 {
		t.Fatalf("marked %v as today, want exactly one day", marked)
	}
	if marked[0] != "2026-08-19" {
		t.Errorf("marked %s as today, want 2026-08-19 — the user's date, not UTC's", marked[0])
	}
}

func TestCellLabelSaysWhichDayIsWhich(t *testing.T) {
	d := func(day int) time.Time { return time.Date(2026, 8, day, 0, 0, 0, 0, time.UTC) }

	cases := []struct {
		cell dayCell
		want string
	}{
		{dayCell{Date: d(18), InMonth: true, IsToday: true}, "🔸18"},
		{dayCell{Date: d(20), InMonth: true, HasEvents: true}, "20•"},
		{dayCell{Date: d(20), InMonth: true}, "20"},
		{dayCell{Date: d(2), InMonth: false}, "·2"},
		// Today wins over busy: one square, and "where am I" beats "what is here".
		{dayCell{Date: d(18), InMonth: true, IsToday: true, HasEvents: true}, "🔸18"},
	}
	for _, c := range cases {
		if got := cellLabel(c.cell); got != c.want {
			t.Errorf("%+v → %q, want %q", c.cell, got, c.want)
		}
	}
}

// Every button in the grid must be routed, and must fit. 45 buttons ship in one
// message here, so a single unrouted prefix is 42 dead squares.
func TestMonthGridButtonsAreRoutedAndFit(t *testing.T) {
	labels := make([]string, 42)
	dates := make([]string, 42)
	for i := range labels {
		labels[i] = "88"
		dates[i] = "2026-08-01"
	}
	kb := keyboards.MonthGrid(2026, 8, "Август", labels, dates)

	routed := []string{"cal_prev_", "cal_next_", "cal_day_", "cal_back_", "noop", "main_menu"}
	var seen int
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				t.Fatal("a grid button carries no callback data — Telegram rejects the keyboard")
			}
			seen++
			data := *b.CallbackData
			if len(data) > 64 {
				t.Errorf("%q is %d bytes", data, len(data))
			}
			var known bool
			for _, p := range routed {
				if strings.HasPrefix(data, p) {
					known = true
					break
				}
			}
			if !known {
				t.Errorf("%q matches no routed prefix", data)
			}
		}
	}
	// 3 header + 7 weekday + 42 days + 1 menu.
	if seen != 53 {
		t.Errorf("grid has %d buttons, want 53", seen)
	}
}

func TestMonthGridRowsAreSevenWide(t *testing.T) {
	labels := make([]string, 42)
	dates := make([]string, 42)
	for i := range labels {
		labels[i] = "1"
		dates[i] = "2026-08-01"
	}
	kb := keyboards.MonthGrid(2026, 8, "Август", labels, dates)

	// Rows 2..7 are the weeks. Telegram renders whatever it is given, so a row
	// of six and a row of eight would simply look wrong and never error.
	if len(kb.InlineKeyboard) != 9 {
		t.Fatalf("got %d rows, want 9 (nav + weekdays + 6 weeks + menu)", len(kb.InlineKeyboard))
	}
	for i := 2; i <= 7; i++ {
		if got := len(kb.InlineKeyboard[i]); got != 7 {
			t.Errorf("week row %d has %d cells, want 7", i-1, got)
		}
	}
}

func TestMonthGridIgnoresAShortLabelSlice(t *testing.T) {
	// Defensive: a truncated slice must not panic mid-render and leave the user
	// with no keyboard at all. Fewer weeks is a visible bug; a crash is a dead bot.
	kb := keyboards.MonthGrid(2026, 8, "Август", []string{"1", "2"}, []string{"a", "b"})
	if len(kb.InlineKeyboard) != 3 {
		t.Errorf("got %d rows, want 3 (nav + weekdays + menu)", len(kb.InlineKeyboard))
	}
}

// 🎯 Today asked the API for the local calendar date stamped as UTC, which for
// Moscow is 03:00 to 03:00: three hours of yesterday evening included and the
// last three hours of tonight missing. It reads correctly for most of the day,
// which is exactly why it survived to be found by a calendar written next to it.
func TestDayBoundsCoverTheLocalDayNotTheUTCOne(t *testing.T) {
	loc := mustLoad(t, "Europe/Moscow")
	// Late enough that the naive version and the correct one disagree.
	now := time.Date(2026, 8, 19, 22, 15, 0, 0, loc)

	from, to := dayBounds(now, loc)

	if from != "2026-08-18T21:00:00Z" {
		t.Errorf("from = %s, want 2026-08-18T21:00:00Z (midnight in Moscow)", from)
	}
	if to != "2026-08-19T21:00:00Z" {
		t.Errorf("to = %s, want 2026-08-19T21:00:00Z", to)
	}

	// The event this used to miss: 23:30 tonight, local.
	tonight := time.Date(2026, 8, 19, 23, 30, 0, 0, loc).UTC()
	start, _ := time.Parse(time.RFC3339, from)
	end, _ := time.Parse(time.RFC3339, to)
	if tonight.Before(start) || !tonight.Before(end) {
		t.Errorf("23:30 tonight (%s) falls outside [%s, %s)", tonight.Format(time.RFC3339), from, to)
	}

	// And the one it used to include by mistake: 23:30 yesterday.
	yesterdayLate := time.Date(2026, 8, 18, 23, 30, 0, 0, loc).UTC()
	if !yesterdayLate.Before(start) {
		t.Errorf("yesterday 23:30 (%s) is inside today's range", yesterdayLate.Format(time.RFC3339))
	}
}

func TestDayBoundsAreExactlyOneDayApartAcrossADSTShift(t *testing.T) {
	// Moscow has no DST, so a zone that does is the only way to check the
	// AddDate-on-local-midnight spelling rather than a fixed 24h add.
	loc := mustLoad(t, "Europe/Berlin")
	// 25 October 2026 is the European autumn change: this local day is 25 hours.
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, loc)

	from, to := dayBounds(now, loc)
	start, _ := time.Parse(time.RFC3339, from)
	end, _ := time.Parse(time.RFC3339, to)

	if got := end.Sub(start); got != 25*time.Hour {
		t.Errorf("the range spans %v, want 25h — the local day really is that long", got)
	}
}
