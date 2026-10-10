package parse

import (
	"testing"
	"time"
)

// 🔴 Denis, 20.09, verbatim: «задача повтор пить таблетки» — and the bot made a
// task called «повтор пить таблетки» that repeated never. The event parser had
// understood these words since v0.4.11.2; the task parser had never been told
// about them, so the same sentence meant two different things depending on
// which button it followed.
func TestTaskLineUnderstandsRepeat(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		line, title, rule string
		asked             bool
	}{
		{"пить таблетки каждый день", "пить таблетки", "FREQ=DAILY", false},
		{"ежедневно пить таблетки", "пить таблетки", "FREQ=DAILY", false},
		{"полить цветы через день", "полить цветы", "FREQ=DAILY;INTERVAL=2", false},
		{"отчёт каждую неделю", "отчёт", "FREQ=WEEKLY", false},
		{"оплатить интернет каждый месяц", "оплатить интернет", "FREQ=MONTHLY", false},
		{"раз в 3 дня протереть пыль", "протереть пыль", "FREQ=DAILY;INTERVAL=3", false},
		{"take pills every day", "take pills", "FREQ=DAILY", false},
		// His exact words: a repeat with no frequency is a question to ask, not
		// a word to keep in the title and not a guess to make.
		{"повтор пить таблетки", "пить таблетки", "", true},
		// And a line with none of it is left completely alone.
		{"купить молоко", "купить молоко", "", false},
		// ⚠ «отжаться 12 раз» is twelve push-ups, not a series of twelve.
		{"отжаться 12 раз", "отжаться 12 раз", "", false},
	}
	for _, c := range cases {
		got := ParseTask(c.line, now)
		if got.Title != c.title {
			t.Errorf("%q: title = %q, want %q", c.line, got.Title, c.title)
		}
		if got.Rrule != c.rule {
			t.Errorf("%q: rule = %q, want %q", c.line, got.Rrule, c.rule)
		}
		if got.RepeatAsked != c.asked {
			t.Errorf("%q: RepeatAsked = %v, want %v", c.line, got.RepeatAsked, c.asked)
		}
	}
}

// The repeat must not eat the other markers, nor they it.
func TestTaskRepeatCoexistsWithTheOtherMarkers(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	got := ParseTask("пить таблетки каждый день !1 5м #здоровье", now)

	if got.Title != "пить таблетки" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Rrule != "FREQ=DAILY" {
		t.Errorf("rule = %q", got.Rrule)
	}
	if got.Priority == nil || *got.Priority != 1 {
		t.Errorf("priority lost: %v", got.Priority)
	}
	if got.EstimatedMinutes == nil || *got.EstimatedMinutes != 5 {
		t.Errorf("estimate lost: %v", got.EstimatedMinutes)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "здоровье" {
		t.Errorf("tags lost: %v", got.Tags)
	}
}
