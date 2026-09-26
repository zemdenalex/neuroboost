package parse

import (
	"testing"
	"time"
)

func spanNow() time.Time { return time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC) } // Wednesday

// 🔴 Denis, 16.09: «нельзя создать событие на несколько дней то есть например
// “отпуск с 14.09 по 29.09”». One event over the whole span — his answer 17.09 —
// not sixteen.
func TestAllDaySpans(t *testing.T) {
	cases := map[string]struct {
		title          string
		from, to       int
		fromMon, toMon time.Month
	}{
		"отпуск с 14.10 по 29.10":       {"отпуск", 14, 29, time.October, time.October},
		"отпуск 14.10-29.10":            {"отпуск", 14, 29, time.October, time.October},
		"vacation from 14.10 to 29.10":  {"vacation", 14, 29, time.October, time.October},
		"дача с понедельника по среду":  {"дача", 21, 23, time.September, time.September},
		"поездка с 28.12 по 03.01.2027": {"поездка", 28, 3, time.December, time.January},
	}
	for line, want := range cases {
		p := ParseLine(line, spanNow())
		d := p.Draft
		if p.Title != want.title {
			t.Errorf("%q: title %q, want %q", line, p.Title, want.title)
		}
		if !d.AllDay || d.HasTime {
			t.Errorf("%q: allDay %v hasTime %v — a span of days with no time is all day", line, d.AllDay, d.HasTime)
		}
		if d.Day.Day() != want.from || d.Day.Month() != want.fromMon {
			t.Errorf("%q: starts %s", line, d.Day.Format("02.01.2006"))
		}
		if d.EndDay.IsZero() || d.EndDay.Day() != want.to || d.EndDay.Month() != want.toMon {
			t.Errorf("%q: ends %s", line, d.EndDay.Format("02.01.2006"))
		}
	}
}

// With times, one timed event across the days — the end is an offset from the
// first day, the way EndsAt already composes an end past midnight.
func TestTimedSpan(t *testing.T) {
	p := ParseLine("конференция с 14.10 10:00 по 16.10 18:00", spanNow())
	d := p.Draft
	if p.Title != "конференция" || d.AllDay {
		t.Fatalf("title %q allDay %v", p.Title, d.AllDay)
	}
	if got := d.StartsAt(); got.Day() != 14 || got.Hour() != 10 {
		t.Errorf("starts %s", got)
	}
	if got := d.EndsAt(); got.Day() != 16 || got.Hour() != 18 {
		t.Errorf("ends %s", got)
	}
}

// 🔴 An end before the start is marked, not swapped: the bot does not know
// which of the two was the typo.
func TestBackwardsSpanIsMarked(t *testing.T) {
	p := ParseLine("отпуск с 29.10 по 14.10", spanNow())
	if !p.Draft.IsUncertain(FieldDay) {
		t.Errorf("a span ending before it starts is not marked ⚠")
	}
}

// Not spans: one date after «до», and the end of a repeating series.
func TestNotASpan(t *testing.T) {
	p := ParseLine("сдать отчёт до 01.12", spanNow())
	if !p.Draft.EndDay.IsZero() {
		t.Errorf("«до 01.12» alone became a span to %s", p.Draft.EndDay)
	}
	p = ParseLine("йога каждую неделю с 01.10 до 01.12", spanNow())
	if !p.Draft.EndDay.IsZero() || p.Draft.RRule() != "FREQ=WEEKLY;UNTIL=2026-12-01" {
		t.Errorf("a repeating series became a span: end %s rule %q", p.Draft.EndDay, p.Draft.RRule())
	}
	if p.Draft.Day.Day() != 1 || p.Draft.Day.Month() != time.October {
		t.Errorf("the series starts %s, want 01.10", p.Draft.Day.Format("02.01"))
	}
}
