package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Final review I3: with the API down, the month does not add two more
// requests (settings, day tasks) that fail the same way and each wait.
func TestAMonthWithoutEventsDoesNotAskForDayTasks(t *testing.T) {
	a := &dayAPI{eventsDown: true}
	h, _, chat := dayHandler(t, a)
	now := time.Now()
	h.showMonth(chat, 0, now.Year(), now.Month())
	after := false
	for _, c := range a.calls {
		if c == "GET /api/events" {
			after = true
			continue
		}
		if after && (strings.HasPrefix(c, "GET /api/day-tasks") || c == "GET /api/auth/me") {
			t.Fatalf("read more after the events read failed: %v", a.calls)
		}
	}
	if !after {
		t.Fatalf("the events read never happened: %v", a.calls)
	}
}

// Final review I3: a failed settings read is not repeated on every screen:
// home() draws each error screen, and each would wait on the same outage.
func TestAFailedSettingsReadIsNotRetriedOnEveryScreen(t *testing.T) {
	a := &dayAPI{meDown: true}
	h, _, chat := dayHandler(t, a)
	for i := 0; i < 3; i++ {
		if !h.dayTasksOn(chat) {
			t.Fatal("a failed read hid day tasks")
		}
	}
	n := 0
	for _, c := range a.calls {
		if c == "GET /api/auth/me" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("settings read %d times in a row during an outage, want 1", n)
	}
}

// Final review I4: since D3 a day not taken is not always ⬛ (today waits,
// days before the start stay plain), and day tasks can be switched off. The
// ℹ️ says both, in both languages.
func TestTheDayTasksHelpMatchesD3(t *testing.T) {
	for _, c := range []struct {
		lang      i18n.Lang
		stale, on string
		before    string
	}{
		{i18n.RU, "остаётся ⬛", "выключ", "до первого"},
		{i18n.EN, "stays ⬛", "switch", "before the first"},
	} {
		s := helpText(c.lang, keyboards.HelpDayTasks)
		if strings.Contains(s, c.stale) {
			t.Errorf("%s: still says a day not taken %q", c.lang, c.stale)
		}
		for _, want := range []string{c.on, c.before} {
			if !strings.Contains(strings.ToLower(s), want) {
				t.Errorf("%s: no %q in %q", c.lang, want, s)
			}
		}
	}
}

// Final review M1: «✏️ Дата» was pressed, day tasks were then switched off,
// and a date typed afterwards still pinned the task. Off wins; the flow ends.
func TestATypedDateWhileDayTasksAreOffPinsNothing(t *testing.T) {
	a := &dayAPI{settings: map[string]any{"day_tasks_enabled": false}}
	h, fake, chat := dayHandler(t, a)
	us := h.store.GetOrCreate(chat)
	us.CurrentFlow, us.FlowStep = dayTaskDateFlow, "date"
	us.FlowData = map[string]any{"task": dtOne}
	h.handleDayTaskDate(chat, "пятница")
	if a.called("POST /api/day-tasks") {
		t.Fatalf("pinned while off: %v", a.calls)
	}
	if got := fake.last(t); !strings.Contains(got.Text, "выключены") {
		t.Errorf("no «off» answer: %q", got.Text)
	}
	if us.CurrentFlow != "" {
		t.Errorf("flow left open: %q", us.CurrentFlow)
	}
}

// Final review M7: the web now switches day tasks too (24.09). A cached
// answer older than the TTL is read again; a fresh one is not.
func TestTheDayTasksSwitchCacheExpires(t *testing.T) {
	a := &dayAPI{settings: map[string]any{"day_tasks_enabled": false}}
	h, _, chat := dayHandler(t, a)
	us := h.store.GetOrCreate(chat)
	us.DayTasksOn, us.DayTasksKnown, us.DayTasksAt = true, true, time.Now()
	if !h.dayTasksOn(chat) {
		t.Fatal("a fresh cache was read again")
	}
	us.DayTasksAt = time.Now().Add(-dayTasksTTL - time.Second)
	if h.dayTasksOn(chat) {
		t.Error("a stale cache was trusted: the web switched day tasks off")
	}
}
