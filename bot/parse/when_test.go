package parse

import (
	"testing"
	"time"
)

func hhmm(h, m int) time.Duration {
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

// 🔴 The original defect, kept as a permanent test. "завтра" lives inside
// "завтрак"; the old parser cut it out and produced the title "к" on tomorrow.
func TestZavtrakIsNotZavtra(t *testing.T) {
	toks := Tokenize("10:00 завтрак")
	var d Draft
	recogniseRelativeDay(toks, tuesday15(), &d)
	recogniseTimeRange(toks, &d)

	if d.HasDay {
		t.Errorf("a day was recognised in %q: %s", "10:00 завтрак", d.Day.Format("2 Jan"))
	}
	if got := Title(toks); got != "завтрак" {
		t.Errorf("Title = %q, want %q", got, "завтрак")
	}
	if !d.HasTime || d.Start != hhmm(10, 0) {
		t.Errorf("Start = %v, want 10:00", d.Start)
	}
}

func TestRelativeDays(t *testing.T) {
	cases := []struct {
		line string
		want int // day of September 2026, today being Tuesday the 15th
	}{
		{"сегодня", 15},
		{"завтра", 16},
		{"послезавтра", 17},
		{"вчера", 14},
		{"позавчера", 13},
		{"today", 15},
		{"tomorrow", 16},
		{"yesterday", 14},
	}
	for _, c := range cases {
		toks := Tokenize(c.line)
		var d Draft
		if !recogniseRelativeDay(toks, tuesday15(), &d) {
			t.Errorf("%q: not recognised", c.line)
			continue
		}
		if d.Day.Day() != c.want {
			t.Errorf("%q: got %d, want %d", c.line, d.Day.Day(), c.want)
		}
		if got := Title(toks); got != "" {
			t.Errorf("%q: leftover in title: %q", c.line, got)
		}
	}
}

func TestExplicitDate(t *testing.T) {
	cases := []struct {
		line       string
		wantDay    int
		wantMonth  time.Month
		wantYear   int
		recognised bool
	}{
		{"16.09", 16, time.September, 2026, true},
		{"16.09.2027", 16, time.September, 2027, true},
		// A bare day.month already gone by this year means next year.
		{"03.02", 3, time.February, 2027, true},
		// Not a date: month 30 does not exist. Left for the time recogniser.
		{"09.30", 0, 0, 0, false},
		{"оркестр", 0, 0, 0, false},
	}
	for _, c := range cases {
		toks := Tokenize(c.line)
		var d Draft
		got := recogniseExplicitDate(toks, tuesday15(), &d)
		if got != c.recognised {
			t.Errorf("%q: recognised = %v, want %v", c.line, got, c.recognised)
			continue
		}
		if !c.recognised {
			continue
		}
		if d.Day.Day() != c.wantDay || d.Day.Month() != c.wantMonth || d.Day.Year() != c.wantYear {
			t.Errorf("%q: got %s, want %d %s %d", c.line, d.Day.Format("2 Jan 2006"), c.wantDay, c.wantMonth, c.wantYear)
		}
	}
}

func TestTimeRange(t *testing.T) {
	cases := []struct {
		line      string
		wantStart time.Duration
		wantEnd   time.Duration
		hasEnd    bool
	}{
		{"14:00", hhmm(14, 0), 0, false},
		{"14:00-15:00", hhmm(14, 0), hhmm(15, 0), true},
		{"14:00 - 15:00", hhmm(14, 0), hhmm(15, 0), true},
		{"14:00–15:30", hhmm(14, 0), hhmm(15, 30), true},
		{"9:05", hhmm(9, 5), 0, false},
		{"14.00", hhmm(14, 0), 0, false},
		{"23:00-01:00", hhmm(23, 0), hhmm(1, 0), true},
	}
	for _, c := range cases {
		toks := Tokenize(c.line)
		var d Draft
		if !recogniseTimeRange(toks, &d) {
			t.Errorf("%q: not recognised", c.line)
			continue
		}
		if d.Start != c.wantStart {
			t.Errorf("%q: Start = %v, want %v", c.line, d.Start, c.wantStart)
		}
		if d.HasEnd != c.hasEnd {
			t.Errorf("%q: HasEnd = %v, want %v", c.line, d.HasEnd, c.hasEnd)
		}
		if c.hasEnd && d.End != c.wantEnd {
			t.Errorf("%q: End = %v, want %v", c.line, d.End, c.wantEnd)
		}
		if got := Title(toks); got != "" {
			t.Errorf("%q: leftover in title: %q", c.line, got)
		}
	}
}

func TestTimeRangeRejectsImpossibleClock(t *testing.T) {
	for _, line := range []string{"25:00", "14:73", "99:99"} {
		toks := Tokenize(line)
		var d Draft
		if recogniseTimeRange(toks, &d) {
			t.Errorf("%q: accepted as a time", line)
		}
		if got := Title(toks); got != line {
			t.Errorf("%q: title changed to %q", line, got)
		}
	}
}

// Denis, item 1: "полдень и тд надо чтобы правильно ставил".
func TestTimeWords(t *testing.T) {
	cases := []struct {
		line string
		want time.Duration
	}{
		{"полдень", hhmm(12, 0)},
		{"полночь", 0},
		{"утром", hhmm(9, 0)},
		{"вечером", hhmm(19, 0)},
		{"ночью", hhmm(22, 0)},
		{"noon", hhmm(12, 0)},
		{"midnight", 0},
		{"morning", hhmm(9, 0)},
		{"evening", hhmm(19, 0)},
	}
	for _, c := range cases {
		toks := Tokenize(c.line)
		var d Draft
		if !recogniseTimeWord(toks, &d) {
			t.Errorf("%q: not recognised", c.line)
			continue
		}
		if d.Start != c.want {
			t.Errorf("%q: Start = %v, want %v", c.line, d.Start, c.want)
		}
		if d.HasEnd {
			t.Errorf("%q: a bare time word must not set an end", c.line)
		}
	}
}

// 🔴 Digits beat words. "полдень 14:00" is 14:00, not noon — and the
// recogniser order is what guarantees it, so this test is about order, not
// about either recogniser on its own.
func TestDigitsWinOverTimeWords(t *testing.T) {
	toks := Tokenize("полдень 14:00 оркестр")
	var d Draft
	recogniseTimeRange(toks, &d)
	recogniseTimeWord(toks, &d)

	if d.Start != hhmm(14, 0) {
		t.Errorf("Start = %v, want 14:00", d.Start)
	}
	// The word is still consumed — it is a time word, so it must not wash up
	// in the title even when a more precise time won.
	if got := Title(toks); got != "оркестр" {
		t.Errorf("Title = %q, want %q", got, "оркестр")
	}
}
