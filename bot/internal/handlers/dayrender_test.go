package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// «Задачи дня» D2: the text of the day screen. Pure — the handler fetches,
// this draws (plan 2026-09-23 Task 2).

var (
	wed23 = time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC) // a Wednesday
	tue22 = wed23.AddDate(0, 0, -1)
)

func TestATakenDayShowsItsColourAndCount(t *testing.T) {
	d := api.Day{Day: "2026-09-23", Target: 5, Confirmed: true, Done: 3, Level: 3,
		Items: []api.DayItem{{TaskID: "a", Title: "отчёт", Done: true}}}
	got := renderDayScreen(i18n.RU, d, nil, wed23, wed23)
	for _, want := range []string{"📌", "Задачи дня", "ср 23.09", "🟧", "3 из 5"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Wed") || strings.Contains(got, "We ") {
		t.Errorf("English weekday in a Russian screen:\n%s", got)
	}
}

func TestAnUntakenDayOffersTheProposal(t *testing.T) {
	d := api.Day{Day: "2026-09-23", Target: 5}
	proposal := []api.DayItem{{TaskID: "a", Title: "отчёт"}, {TaskID: "b", Title: "позвонить маме"}}
	got := renderDayScreen(i18n.RU, d, proposal, wed23, wed23)
	for _, want := range []string{"День ещё не взят", "отчёт", "позвонить маме"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// Review focus 2: nothing open to take — say so, do not offer an empty «Беру».
func TestAnUntakenDayWithNothingToTakeSaysSo(t *testing.T) {
	got := renderDayScreen(i18n.RU, api.Day{Day: "2026-09-23", Target: 5}, nil, wed23, wed23)
	if !strings.Contains(got, "Взять нечего") {
		t.Errorf("an empty proposal does not say so:\n%s", got)
	}
}

// Denis 22.09: a day never taken is ⬛, «не выбрал = не сделал».
func TestAPastUntakenDayIsBlack(t *testing.T) {
	got := renderDayScreen(i18n.RU, api.Day{Day: "2026-09-22", Target: 5}, nil, tue22, wed23)
	if !strings.Contains(got, "⬛") || !strings.Contains(got, "День не был взят") {
		t.Errorf("a past untaken day:\n%s", got)
	}
}

func TestTheDayScreenInEnglish(t *testing.T) {
	d := api.Day{Day: "2026-09-23", Target: 5, Confirmed: true, Done: 3, Level: 3}
	got := renderDayScreen(i18n.EN, d, nil, wed23, wed23)
	if !strings.Contains(got, "3 of 5") || !strings.Contains(got, "Day tasks") {
		t.Errorf("English screen:\n%s", got)
	}
}
