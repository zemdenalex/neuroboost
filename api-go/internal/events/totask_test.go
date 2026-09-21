package events

import (
	"reflect"
	"testing"
	"time"
)

func strp(s string) *string { return &s }

// Spec A3: «Событие 14:50–16:00 → задача → срок = тот день, оценка = 1ч10м,
// карточка говорила ⚠ у задачи нет часа».
func TestAnEventBecomesATaskOnItsLocalDay(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	start := time.Date(2026, 10, 20, 14, 50, 0, 0, msk)
	ev := Event{Title: "врач", CalendarID: "c1", Tags: []string{"здоровье"},
		StartsAt: start, EndsAt: start.Add(70 * time.Minute), Color: strp("#f00")}

	got, lost := taskFromEvent(ev, ev.StartsAt, ev.EndsAt, "Europe/Moscow", false)

	if got.DueDate != "2026-10-20" {
		t.Errorf("due = %s, want 2026-10-20", got.DueDate)
	}
	if got.EstimatedMinutes == nil || *got.EstimatedMinutes != 70 {
		t.Errorf("estimate = %v, want 70", got.EstimatedMinutes)
	}
	if want := []string{LostStartTime, LostColor}; !reflect.DeepEqual(lost, want) {
		t.Errorf("lost = %v, want %v", lost, want)
	}
}

// 🔴 «Чьё это сегодня»: 01:30 в Москве — это ещё вчера по UTC. Срок обязан
// быть московским днём, а не днём сервера.
func TestTheDueDayIsTheUsersNotTheServers(t *testing.T) {
	start := time.Date(2026, 10, 19, 22, 30, 0, 0, time.UTC) // 20.10 01:30 MSK
	ev := Event{Title: "x", StartsAt: start, EndsAt: start.Add(time.Hour)}
	got, _ := taskFromEvent(ev, start, start.Add(time.Hour), "Europe/Moscow", false)
	if got.DueDate != "2026-10-20" {
		t.Errorf("due = %s, want the Moscow day 2026-10-20", got.DueDate)
	}
}

func TestAnAllDayEventLosesNoHourAndHasNoEstimate(t *testing.T) {
	start := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	ev := Event{Title: "отпуск", AllDay: true, StartsAt: start, EndsAt: start.AddDate(0, 0, 1),
		Location: strp("Сочи"), ReminderOffsets: []int{60}}
	got, lost := taskFromEvent(ev, ev.StartsAt, ev.EndsAt, "UTC", false)
	if got.EstimatedMinutes != nil {
		t.Errorf("all-day estimate = %v, want none", *got.EstimatedMinutes)
	}
	if want := []string{LostLocation, LostReminders}; !reflect.DeepEqual(lost, want) {
		t.Errorf("lost = %v, want %v", lost, want)
	}
}

func TestOnlyASeriesCarriesTheRule(t *testing.T) {
	start := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	ev := Event{Title: "x", StartsAt: start, EndsAt: start.Add(time.Hour), Rrule: strp("FREQ=WEEKLY")}
	series, _ := taskFromEvent(ev, start, start.Add(time.Hour), "UTC", true)
	once, _ := taskFromEvent(ev, start, start.Add(time.Hour), "UTC", false)
	if series.Rrule == nil || *series.Rrule != "FREQ=WEEKLY" {
		t.Errorf("series rrule = %v", series.Rrule)
	}
	if once.Rrule != nil {
		t.Errorf("once carried the rule: %v", *once.Rrule)
	}
}
