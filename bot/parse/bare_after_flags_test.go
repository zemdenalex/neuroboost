package parse

import "testing"

// 🔴 Denis, 27.10: «any number in the start (and start counts as any non-title
// flag or tag) or end counts in time or date … number after a day is hour».
// A tag, «задача» or a colour in front of the number is not title text, so the
// number still stands at the start of what the user wrote. Before this, the tag,
// kind and colour passes ran AFTER the time pass, and «#работа завтра 10 врач»
// became the task «10 врач» with no time.
func TestBareHourAfterAFlagIsStillAtTheStart(t *testing.T) {
	cases := []struct{ line, title string }{
		{"#работа завтра 10 врач", "врач"},
		{"задача завтра 10 врач", "врач"},
		{"синий завтра 10 врач", "врач"},
	}
	for _, c := range cases {
		p := ParseLine(c.line, tuesday15())
		if !p.Draft.HasTime || p.Draft.Start != hhmm(10, 0) {
			t.Errorf("%q: start = %v (has time %v), want 10:00", c.line, p.Draft.Start, p.Draft.HasTime)
		}
		if p.Title != c.title {
			t.Errorf("%q: Title = %q, want %q", c.line, p.Title, c.title)
		}
		if !p.Draft.IsUncertain(FieldTime) {
			t.Errorf("%q: a bare hour is a loose reading and must be marked", c.line)
		}
	}
}

// The end of the line counts too, but only when the line names a day: «врач
// завтра 10» is an appointment, «кабинет 12» (no day) stays a room number —
// TestBareNumberIsNotATimeInTheMiddleOfAPhrase holds that side.
func TestBareHourAtTheEndCountsWhenADayIsNamed(t *testing.T) {
	p := ParseLine("врач завтра 10", tuesday15())
	if !p.Draft.HasTime || p.Draft.Start != hhmm(10, 0) {
		t.Fatalf("start = %v (has time %v), want 10:00", p.Draft.Start, p.Draft.HasTime)
	}
	if p.Title != "врач" {
		t.Errorf("Title = %q, want «врач»", p.Title)
	}
	if !p.Draft.IsUncertain(FieldTime) {
		t.Error("a bare hour is a loose reading and must be marked")
	}

	// Positive control for the guard: the same shape without a day is left alone.
	if q := ParseLine("купить яйца 10", tuesday15()); q.Draft.HasTime {
		t.Errorf("«купить яйца 10»: read %v as a time with no day in the line", q.Draft.Start)
	}
}
