package keyboards

import "testing"

func TestDayActionsFitTheCallbackBudget(t *testing.T) {
	kb := DayActions("2026-08-18", "2026-08-17", "2026-08-19", 2026, 8)
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
		}
	}
}

func TestMonthGridOffersAJumpToToday(t *testing.T) {
	kb := MonthGrid(2026, 8, "август", make([]string, 42), make([]string, 42), "2026-08-18")
	var found bool
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil && *b.CallbackData == "cal_day_2026-08-18" {
				found = true
			}
		}
	}
	if !found {
		t.Error("the month grid has no jump to today — paging back from December is the only way home")
	}
}
