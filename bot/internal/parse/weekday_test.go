package parse

import (
	"testing"
	"time"
)

// tuesday15 is Tuesday 15.09.2026, 19:00 local — the day Denis wrote the notes,
// and the day every example in his list is relative to.
func tuesday15() time.Time {
	return time.Date(2026, 9, 15, 19, 0, 0, 0, time.UTC)
}

// Denis's rule, 15.09: a weekday with no modifier is the NEAREST one and that
// includes today; "следующая" is that same nearest day plus seven.
//
// The pair that decides it: on a Wednesday, "среда" is today and "следующая
// среда" is a week out — not "next Wednesday means the one after the one I'd
// have called this Wednesday".
func TestWeekdayResolution(t *testing.T) {
	cases := []struct {
		line    string
		wantDay int // day of September 2026
	}{
		// Today is Tuesday 15.09.
		{"среда", 16},
		{"следующая среда", 23},
		{"эта среда", 16},
		{"вторник", 15},           // today
		{"следующий вторник", 22}, // today + 7
		{"понедельник", 21},
		{"воскресенье", 20},
		{"прошлая среда", 9},

		// Declined forms — people write "в среду", not "в среда".
		{"в среду", 16},
		{"в пятницу", 18},
		{"в субботу", 19},

		// Russian abbreviations, Denis's choice 15.09.
		{"ср", 16},
		{"пн", 21},
		{"вс", 20},

		// English, full and abbreviated.
		{"wednesday", 16},
		{"next wednesday", 23},
		{"wed", 16},
		{"mon", 21},
		{"sun", 20},
		{"on friday", 18},
	}

	for _, c := range cases {
		toks := Tokenize(c.line)
		var d Draft
		if !recogniseWeekday(toks, tuesday15(), &d) {
			t.Errorf("%q: not recognised", c.line)
			continue
		}
		if d.Day.Day() != c.wantDay || d.Day.Month() != time.September {
			t.Errorf("%q: got %s, want %d September", c.line, d.Day.Format("2 January"), c.wantDay)
		}
		if got := Title(toks); got != "" {
			t.Errorf("%q: leftover in title: %q", c.line, got)
		}
	}
}

// 🔴 The reason this whole file exists. A weekday word that lives inside
// another word is not a weekday word.
func TestWeekdayNeverMatchesInsideAnotherWord(t *testing.T) {
	for _, line := range []string{"ужин втроём", "pay the money", "sunny walk", "срочно"} {
		toks := Tokenize(line)
		var d Draft
		if recogniseWeekday(toks, tuesday15(), &d) {
			t.Errorf("%q: recognised a weekday that is not there (got %s)", line, d.Day)
		}
		if got := Title(toks); got != line {
			t.Errorf("%q: title changed to %q", line, got)
		}
	}
}

// The day is claimed; everything else stays for the title. This is the whole
// contract between the recogniser and Title.
func TestWeekdayLeavesTheRestOfTheTitleAlone(t *testing.T) {
	toks := Tokenize("среда оркестр")
	var d Draft
	if !recogniseWeekday(toks, tuesday15(), &d) {
		t.Fatal("not recognised")
	}
	if got := Title(toks); got != "оркестр" {
		t.Errorf("Title = %q, want %q", got, "оркестр")
	}
}

// A modifier with no weekday after it is not a day. "Следующий раз" is a title.
func TestModifierAloneIsNotADay(t *testing.T) {
	toks := Tokenize("следующий раз")
	var d Draft
	if recogniseWeekday(toks, tuesday15(), &d) {
		t.Error("recognised a day in «следующий раз»")
	}
	if got := Title(toks); got != "следующий раз" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}

// Midnight, in the clock's own location — a day is a date, not an instant.
func TestWeekdayReturnsLocalMidnight(t *testing.T) {
	toks := Tokenize("среда")
	var d Draft
	if !recogniseWeekday(toks, tuesday15(), &d) {
		t.Fatal("not recognised")
	}
	if h, m, s := d.Day.Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("got %s, want midnight", d.Day.Format("15:04:05"))
	}
}
