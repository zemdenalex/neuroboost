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
