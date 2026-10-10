package parse

import (
	"testing"
	"time"
)

func repeatNow() time.Time { return time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC) }

// 🔴 Denis, 16.09: «кому-то нужно пить таблетки раз в 3 дня, или кормить змея
// раз в 3 недели, или что-то с необычной периодичностью».
func TestCustomRepeatPeriods(t *testing.T) {
	cases := map[string]string{
		"таблетки раз в 3 дня 09:00":  "FREQ=DAILY;INTERVAL=3",
		"кормить змея раз в 3 недели": "FREQ=WEEKLY;INTERVAL=3",
		"оплата раз в месяц":          "FREQ=MONTHLY",
		"полив каждые 2 дня":          "FREQ=DAILY;INTERVAL=2",
		"gym every 2 weeks":           "FREQ=WEEKLY;INTERVAL=2",
		"уборка через день":           "FREQ=DAILY;INTERVAL=2",
		"зарядка каждый день 07:00":   "FREQ=DAILY",
		"отчёт раз в 2 месяца":        "FREQ=MONTHLY;INTERVAL=2",
		"техосмотр раз в 2 года":      "FREQ=MONTHLY;INTERVAL=24",
	}
	for line, want := range cases {
		p := ParseLine(line, repeatNow())
		if got := p.Draft.RRule(); got != want {
			t.Errorf("%q: RRULE %q, want %q (title %q)", line, got, want, p.Title)
		}
	}
}

// 🔴 The API understands FREQ=DAILY|WEEKLY|MONTHLY with INTERVAL, COUNT, UNTIL
// and NOTHING else (api-go/internal/events/recurrence.go). An RRULE it cannot
// parse is stored, and the event is shown once and never repeats — silently.
// So «каждый год» and «каждый вторник» must be written in words it knows.
func TestRepeatOnlyUsesWhatTheAPIParses(t *testing.T) {
	cases := map[string]string{
		"день рождения каждый год": "FREQ=MONTHLY;INTERVAL=12",
		"налоги ежегодно":          "FREQ=MONTHLY;INTERVAL=12",
		"оркестр каждый вторник":   "FREQ=WEEKLY",
	}
	for line, want := range cases {
		p := ParseLine(line, repeatNow())
		if got := p.Draft.RRule(); got != want {
			t.Errorf("%q: RRULE %q, want %q", line, got, want)
		}
	}
	// And the weekday still chooses the first day.
	p := ParseLine("оркестр каждый вторник", repeatNow())
	if !p.Draft.HasDay || p.Draft.Day.Weekday() != time.Tuesday {
		t.Errorf("«каждый вторник» starts on %s", p.Draft.Day.Weekday())
	}
}

// How many times, and until when — read only when the line repeats.
func TestRepeatEnds(t *testing.T) {
	cases := map[string]string{
		"таблетки раз в 3 дня 10 раз":     "FREQ=DAILY;INTERVAL=3;COUNT=10",
		"йога каждую неделю до 01.12":     "FREQ=WEEKLY;UNTIL=2026-12-01",
		"gym every 2 weeks 8 times":       "FREQ=WEEKLY;INTERVAL=2;COUNT=8",
		"курс ежедневно until 30.09.2026": "FREQ=DAILY;UNTIL=2026-09-30",
	}
	for line, want := range cases {
		p := ParseLine(line, repeatNow())
		if got := p.Draft.RRule(); got != want {
			t.Errorf("%q: RRULE %q, want %q (title %q)", line, got, want, p.Title)
		}
	}
}

// 🔴 Without a repeat, «12 раз» is part of the title and «до 01.12» is a date
// — not the end of a series that does not exist.
func TestRepeatEndWordsNeedARepeat(t *testing.T) {
	p := ParseLine("отжаться 12 раз", repeatNow())
	if p.Title != "отжаться 12 раз" || p.Draft.RRule() != "" {
		t.Errorf("«отжаться 12 раз» → title %q, rrule %q", p.Title, p.Draft.RRule())
	}
	p = ParseLine("сдать отчёт до 01.12", repeatNow())
	if p.Draft.RRule() != "" || !p.Draft.HasDay || p.Draft.Day.Month() != time.December {
		t.Errorf("«до 01.12» without a repeat → rrule %q, day %v", p.Draft.RRule(), p.Draft.Day)
	}
}

// «повтор 10 раз» — the count is kept while the frequency is still to be asked,
// and joins it once chosen.
func TestRepeatCountWaitsForTheFrequency(t *testing.T) {
	p := ParseLine("созвон повтор 10 раз", repeatNow())
	if !p.Draft.RepeatAsked || p.Draft.RepeatCount != 10 {
		t.Fatalf("asked %v count %d", p.Draft.RepeatAsked, p.Draft.RepeatCount)
	}
	d := p.Draft
	d.Repeat = "FREQ=DAILY"
	if d.RRule() != "FREQ=DAILY;COUNT=10" {
		t.Errorf("RRULE after choosing = %q", d.RRule())
	}
}

// SplitRRule is the way back, for an event opened to edit.
func TestSplitRRuleRoundTrips(t *testing.T) {
	for _, rule := range []string{"FREQ=DAILY", "FREQ=WEEKLY;INTERVAL=2;COUNT=8", "FREQ=DAILY;UNTIL=2026-09-30", "FREQ=MONTHLY;INTERVAL=12"} {
		var d Draft
		SplitRRule(rule, &d)
		if d.RRule() != rule {
			t.Errorf("SplitRRule(%q) → %q", rule, d.RRule())
		}
	}
}

func TestRepeatText(t *testing.T) {
	d, ok := RepeatText("раз в 3 дня 10 раз", repeatNow())
	if !ok || d.RRule() != "FREQ=DAILY;INTERVAL=3;COUNT=10" {
		t.Errorf("RepeatText → %q, %v", d.RRule(), ok)
	}
	if _, ok := RepeatText("позвонить маме", repeatNow()); ok {
		t.Errorf("text with no repeat in it was read as one")
	}
}

func TestRepeatEndText(t *testing.T) {
	var d Draft
	d.Repeat = "FREQ=DAILY"
	if !RepeatEndText("5 раз", repeatNow(), &d) || d.RRule() != "FREQ=DAILY;COUNT=5" {
		t.Errorf("«5 раз» → %q", d.RRule())
	}
	if !RepeatEndText("до 20.10", repeatNow(), &d) || d.RRule() != "FREQ=DAILY;UNTIL=2026-10-20" {
		t.Errorf("«до 20.10» → %q — a date end replaces a count", d.RRule())
	}
	if RepeatEndText("когда-нибудь", repeatNow(), &d) {
		t.Errorf("nonsense read as an end")
	}
}
