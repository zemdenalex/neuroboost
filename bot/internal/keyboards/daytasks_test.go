package keyboards

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// «Задачи дня» D2 keyboards (plan 2026-09-23 Task 3).

const dtID = "11111111-2222-3333-4444-555555555555"

func datas(kb tgbotapi.InlineKeyboardMarkup) []string {
	var out []string
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil {
				out = append(out, *b.CallbackData)
			}
		}
	}
	return out
}

func has(kb tgbotapi.InlineKeyboardMarkup, prefix string) bool {
	for _, d := range datas(kb) {
		if strings.HasPrefix(d, prefix) {
			return true
		}
	}
	return false
}

func dayKeyboards() map[string]tgbotapi.InlineKeyboardMarkup {
	items := []DayButton{{ID: dtID, Title: "отчёт"}, {ID: dtID, Title: "молоко", Done: true}}
	v := DayView{Day: "2026-09-23", Prev: "2026-09-22", Next: "2026-09-24", Items: items}
	taken := v
	taken.Taken, taken.Today = true, true
	return map[string]tgbotapi.InlineKeyboardMarkup{
		"screen-taken": DayScreen(i18n.RU, taken),
		"screen-offer": DayScreen(i18n.RU, DayView{Day: v.Day, Prev: v.Prev, Next: v.Next, CanTake: true}),
		"edit":         DayEdit(i18n.RU, "2026-09-23", items, false),
		"add":          DayAddList(i18n.RU, "2026-09-23", items),
		"pin":          DayPinPick(i18n.RU, dtID, "2026-09-23", "2026-09-24"),
		"pinned":       DayPinned(i18n.RU, dtID, "2026-09-23"),
		"settings-on":  DaySettings(i18n.RU, DaySettingsView{On: true, Target: 5}),
		"settings-off": DaySettings(i18n.RU, DaySettingsView{}),
		"target-pick":  DayTargetPick(i18n.RU, 5, "ob_dtn_", "ob_finish", "Дальше →"),
		"date-cancel":  DayDateCancel(i18n.RU, dtID),
		"off":          DayTasksOff(i18n.RU),
	}
}

// Every day-tasks screen explains itself (the help scan asks for it anyway),
// and every callback fits Telegram's 64 bytes.
func TestDayKeyboardsHaveHelpAndFit(t *testing.T) {
	for name, kb := range dayKeyboards() {
		if !has(kb, "help_"+HelpDayTasks) {
			t.Errorf("%s has no ℹ️", name)
		}
		for _, d := range datas(kb) {
			if len(d) > 64 {
				t.Errorf("%s: %q is %d bytes", name, d, len(d))
			}
		}
	}
}

// Today: an undone task is ticked by a press (Denis 23.09: «Mark it done»);
// a done one opens its card.
func TestTodaysTasksAreTickedByAPress(t *testing.T) {
	kb := dayKeyboards()["screen-taken"]
	ds := strings.Join(datas(kb), " ")
	if !strings.Contains(ds, "dt_ok_2026-09-23_"+dtID) || !strings.Contains(ds, "task_action_"+dtID) {
		t.Errorf("today's buttons: %s", ds)
	}
	var labels []string
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			labels = append(labels, b.Text)
		}
	}
	if l := strings.Join(labels, "|"); !strings.Contains(l, "⬜ отчёт") || !strings.Contains(l, "✅ молоко") {
		t.Errorf("labels: %s", l)
	}
	if has(kb, "dt_take_") {
		t.Errorf("a taken day still offers «Беру»")
	}
}

// Another day: no ticking from here, every task opens its card.
func TestAnotherDaysTasksOpenTheirCards(t *testing.T) {
	v := DayView{Day: "2026-09-24", Prev: "2026-09-23", Next: "2026-09-25", Taken: true,
		Items: []DayButton{{ID: dtID, Title: "отчёт"}}}
	if kb := DayScreen(i18n.RU, v); has(kb, "dt_ok_") || !has(kb, "task_action_") {
		t.Errorf("another day's buttons: %v", datas(kb))
	}
}

// Denis 23.09: an untaken day offers «✅ Беру» and «✏️ Выбрать самому».
func TestAnUntakenDayOffersBothWays(t *testing.T) {
	kb := dayKeyboards()["screen-offer"]
	if !has(kb, "dt_take_2026-09-23") || !has(kb, "dt_edit_2026-09-23") {
		t.Errorf("untaken day: %v", datas(kb))
	}
}

// Nothing to take: no empty «Беру», only building the set by hand.
func TestNothingToTakeHasNoTakeButton(t *testing.T) {
	kb := DayScreen(i18n.RU, DayView{Day: "2026-09-23", Prev: "a", Next: "b"})
	if has(kb, "dt_take_") || !has(kb, "dt_edit_") {
		t.Errorf("empty offer: %v", datas(kb))
	}
}

// A past day is read only.
func TestAPastDayCannotBeEdited(t *testing.T) {
	kb := DayScreen(i18n.RU, DayView{Day: "2026-09-22", Prev: "a", Next: "b", Past: true, Taken: true})
	if has(kb, "dt_edit_") || has(kb, "dt_take_") || !has(kb, "dt_d_a") || !has(kb, "dt_d_b") {
		t.Errorf("past day: %v", datas(kb))
	}
}

func TestEditTakesTheSetOnlyWhenNotTaken(t *testing.T) {
	items := []DayButton{{ID: dtID, Title: "отчёт"}}
	if kb := DayEdit(i18n.RU, "2026-09-23", items, false); !has(kb, "dt_takeset_") || !has(kb, "dt_rm_2026-09-23_"+dtID) || !has(kb, "dt_add_") {
		t.Errorf("edit, untaken: %v", datas(kb))
	}
	if kb := DayEdit(i18n.RU, "2026-09-23", items, true); has(kb, "dt_takeset_") {
		t.Errorf("edit, taken, still offers take: %v", datas(kb))
	}
	if kb := DayEdit(i18n.RU, "2026-09-23", nil, false); has(kb, "dt_takeset_") {
		t.Errorf("edit, empty set, offers to take nothing: %v", datas(kb))
	}
}

// Denis 23.09: the day for a task is today, tomorrow, or a typed date.
func TestPinOffersTodayTomorrowAndADate(t *testing.T) {
	kb := DayPinPick(i18n.RU, dtID, "2026-09-23", "2026-09-24")
	for _, want := range []string{"dt_pd_2026-09-23_" + dtID, "dt_pd_2026-09-24_" + dtID, "dt_pdt_" + dtID, "task_action_" + dtID} {
		if !has(kb, want) {
			t.Errorf("pin pick has no %s: %v", want, datas(kb))
		}
	}
}

func TestTargetOffersThreeToSevenAndTicksTheCurrent(t *testing.T) {
	kb := DaySettings(i18n.RU, DaySettingsView{On: true, Target: 5})
	for n := 3; n <= 7; n++ {
		if !has(kb, "dtn_"+string(rune('0'+n))) {
			t.Errorf("no dtn_%d", n)
		}
	}
	found := false
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.Text == "✓ 5" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("the current target is not ticked")
	}
}

// The ways in (spec §8): the menu, a task's card, and the target in Settings.
func TestDayTasksCanBeReached(t *testing.T) {
	if !has(HomeInlineFor(i18n.RU, true), "dt_d_today") {
		t.Errorf("the menu has no 📌 Задачи дня")
	}
	if !has(TaskActions(i18n.RU, dtID, false, "", true), "dt_pin_"+dtID) {
		t.Errorf("the task card has no 📌 В задачи дня")
	}
	// Spec §11: switched off, the card has no 📌.
	if has(TaskActions(i18n.RU, dtID, false, "", false), "dt_pin_"+dtID) {
		t.Errorf("day tasks off, and the card still has 📌")
	}
	if !has(SettingsMenu(i18n.RU), "settings_dtn") {
		t.Errorf("Settings has no 🎯 Задач в день")
	}
}
