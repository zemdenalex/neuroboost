package parse

import "testing"

func TestCalendarByBareName(t *testing.T) {
	toks := Tokenize("созвон работа 14:00")
	recogniseTimeRange(toks, &Draft{})
	var d Draft
	idx, ok := RecogniseCalendar(toks, []string{"Личный", "Работа"}, &d)
	if !ok || idx != 1 {
		t.Fatalf("idx = %d, ok = %v, want 1, true", idx, ok)
	}
	if d.Calendar != "Работа" {
		t.Errorf("Calendar = %q, want %q", d.Calendar, "Работа")
	}
}

func TestCalendarByOpener(t *testing.T) {
	toks := Tokenize("созвон календарь работа")
	var d Draft
	if _, ok := RecogniseCalendar(toks, []string{"Работа"}, &d); !ok {
		t.Fatal("not recognised")
	}
	if got := Title(toks); got != "созвон" {
		t.Errorf("Title = %q, want %q — the opener must be claimed too", got, "созвон")
	}
}

// 🔴 Exact match only. A calendar named «Работа» must not turn the word
// «работа» inside a title into a calendar change whenever it is part of a
// longer word or a different word entirely.
func TestCalendarMatchIsExact(t *testing.T) {
	for _, line := range []string{"созвон работать", "созвон подработка", "созвон раб"} {
		toks := Tokenize(line)
		var d Draft
		if _, ok := RecogniseCalendar(toks, []string{"Работа"}, &d); ok {
			t.Errorf("%q: matched the calendar «Работа»", line)
		}
		if got := Title(toks); got != line {
			t.Errorf("%q: title changed to %q", line, got)
		}
	}
}

func TestCalendarUnknownNameIsLeftAlone(t *testing.T) {
	toks := Tokenize("созвон дача")
	var d Draft
	if _, ok := RecogniseCalendar(toks, []string{"Личный", "Работа"}, &d); ok {
		t.Error("matched a calendar that does not exist")
	}
	if got := Title(toks); got != "созвон дача" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}

// An opener followed by a name nobody has is not a calendar — and the words
// stay in the title rather than vanishing.
func TestCalendarOpenerWithUnknownNameKeepsTheWords(t *testing.T) {
	toks := Tokenize("созвон календарь дача")
	var d Draft
	if _, ok := RecogniseCalendar(toks, []string{"Работа"}, &d); ok {
		t.Error("matched an unknown calendar")
	}
	if got := Title(toks); got != "созвон календарь дача" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}
