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

func TestTaskActionsOfferTheOptionalFields(t *testing.T) {
	id := "11111111-2222-3333-4444-555555555555" // 36 chars, a real UUID length
	kb := TaskActions(id)
	want := map[string]bool{
		"task_sched_" + id: false,
		"task_due_" + id:   false,
		"task_est_" + id:   false,
		"task_tag_" + id:   false,
		"task_done_" + id:  false,
	}
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
			if _, ok := want[*b.CallbackData]; ok {
				want[*b.CallbackData] = true
			}
		}
	}
	for data, found := range want {
		if !found {
			t.Errorf("the task card has no button for %q", data)
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
