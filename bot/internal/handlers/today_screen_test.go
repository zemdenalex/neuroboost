package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// The «Сегодня» screen as Настя read it on 21.09 — the first person to look at
// this bot without knowing how it was built.
//
//	🎯 Сегодня — Mon, Sep 21      ← English, with everything around it Russian
//	📅 События: 2                 ← «не сразу понятно, что значат эти цифры»
//	  14:50 — Оркестр             ← «И почему он границу во времени не пишет?»
//	🎯 Задачи: 6
//	  …five lines…                ← the sixth silently missing
func TestTheDayIsNamedInTheReadersLanguage(t *testing.T) {
	monday := time.Date(2026, time.September, 21, 14, 6, 0, 0, time.UTC)

	if got := dayLabel(i18n.RU, monday); got != "пн, 21 сентября" {
		t.Errorf("RU = %q, want «пн, 21 сентября»", got)
	}
	if got := dayLabel(i18n.EN, monday); got != "Mon, Sep 21" {
		t.Errorf("EN = %q, want «Mon, Sep 21»", got)
	}

	// 🔴 The shape of the bug, asserted directly: Go's month and day names
	// must not appear in a Russian screen.
	ru := dayLabel(i18n.RU, monday)
	for _, english := range []string{"Mon", "Sep", "Monday", "September"} {
		if strings.Contains(ru, english) {
			t.Errorf("Russian label %q still carries %q", ru, english)
		}
	}

	// Every weekday, so the Sunday-vs-Monday indexing cannot be wrong on one
	// day only — time.Weekday starts at Sunday, weekdayShort at Monday.
	want := []string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
	for i := 0; i < 7; i++ {
		d := time.Date(2026, time.September, 20+i, 12, 0, 0, 0, time.UTC) // 20.09 is a Sunday
		if got := dayLabel(i18n.RU, d); !strings.HasPrefix(got, want[i]) {
			t.Errorf("%s: label %q, want it to start with %q", d.Weekday(), got, want[i])
		}
	}
}

func TestAnEventSaysWhenItEnds(t *testing.T) {
	h, _ := newTestHandler(t)

	both := h.eventWhen(1, api.Event{
		StartsAt: "2026-09-21T11:50:00Z",
		EndsAt:   "2026-09-21T13:00:00Z",
	})
	if !strings.Contains(both, "–") {
		t.Errorf("when = %q, want both ends — Настя wrote «со скольки до скольки» and it was in the data", both)
	}

	allDay := h.eventWhen(1, api.Event{
		StartsAt: "2026-09-21T00:00:00Z",
		EndsAt:   "2026-09-22T00:00:00Z",
		AllDay:   true,
	})
	if strings.Contains(allDay, "–") {
		t.Errorf("an all-day event got a time range: %q", allDay)
	}
}

// 🔴 The count and the list must describe the same thing.
func TestTheCountNeverDisagreesWithTheList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/tasks") {
			rows := make([]string, 0, 6)
			for i := 1; i <= 6; i++ {
				rows = append(rows, fmt.Sprintf(`{"id":"t%d","title":"задача %d","status":"TODO","priority":3}`, i, i))
			}
			_, _ = fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(rows, ","))
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	h.handleToday(600, 0)

	text := fake.last(t).Text
	if !strings.Contains(text, "Задачи: 6") {
		t.Fatalf("the header does not carry the count:\n%s", text)
	}

	shown := strings.Count(text, "задача ")
	if shown == 6 {
		return // all six listed: nothing is hidden, nothing to admit
	}
	if !strings.Contains(text, "ещё 1") {
		t.Errorf("header says 6, list shows %d, and the screen never admits the gap:\n%s", shown, text)
	}
}

// A day opened from the calendar shows BOTH halves of what is on it.
//
// 🔴 Настя, 21.09: «Нет функции: Показать задачи и события на день … Очень
// нужна». The screen existed — she had it open — and listed only events, which
// is exactly why she could not recognise it as the thing she wanted.
func TestADayShowsTasksAsWellAsEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/tasks") {
			_, _ = w.Write([]byte(`{"data":[
				{"id":"t1","title":"отдать документы","status":"TODO","priority":2,"due_date":"2026-09-22"},
				{"id":"t2","title":"другой день","status":"TODO","priority":2,"due_date":"2026-09-23"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"e1","title":"Оркестр",
			"starts_at":"2026-09-22T11:50:00Z","ends_at":"2026-09-22T13:00:00Z"}]}`))
	}))
	defer srv.Close()

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	h.handleCalendarDay(601, 0, "2026-09-22")

	text := fake.last(t).Text
	if !strings.Contains(text, "Оркестр") {
		t.Errorf("the day lost its events:\n%s", text)
	}
	if !strings.Contains(text, "отдать документы") {
		t.Errorf("the day still shows no tasks — this is the function Настя could not find:\n%s", text)
	}
	if strings.Contains(text, "другой день") {
		t.Errorf("a task due on another date leaked into this day:\n%s", text)
	}
}

func TestADueDateIsComparedAsADate(t *testing.T) {
	day := time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC)
	tasks := []api.Task{
		{ID: "a", DueDate: "2026-09-22"},
		{ID: "b", DueDate: "2026-09-22T00:00:00Z"},
		{ID: "c", DueDate: "2026-09-21"},
		{ID: "d", DueDate: ""},
	}
	got := tasksDueOn(tasks, day)
	if len(got) != 2 {
		t.Fatalf("matched %d tasks, want 2 (a and b): %+v", len(got), got)
	}
}
