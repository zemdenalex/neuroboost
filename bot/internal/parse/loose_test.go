package parse

import (
	"testing"
	"time"
)

// 🔴 Every one of these came out of Denis's own timetable on 15.09, and four
// of the five were misread. Two of them silently moved a piece of the time into
// the title — «12:00 -12:50 физра» became the event «12:50 физра».
func TestLooseTimeFormsFromHisTimetable(t *testing.T) {
	cases := []struct {
		line      string
		title     string
		start     time.Duration
		end       time.Duration
		hasEnd    bool
		uncertain bool
	}{
		// Strict, and must stay strict — no warning on these.
		{line: "14:50-18:00 оркестр", title: "оркестр", start: hhmm(14, 50), end: hhmm(18, 0), hasEnd: true},
		{line: "9:00 учу сама спец", title: "учу сама спец", start: hhmm(9, 0)},

		// A space before the dash. The end used to land in the title.
		{line: "12:00 -12:50 физра", title: "физра", start: hhmm(12, 0), end: hhmm(12, 50), hasEnd: true, uncertain: true},
		{line: "12:00 -14:40 обед", title: "обед", start: hhmm(12, 0), end: hhmm(14, 40), hasEnd: true, uncertain: true},
		{line: "12:00- 12:50 физра", title: "физра", start: hhmm(12, 0), end: hhmm(12, 50), hasEnd: true, uncertain: true},

		// An end with no minutes.
		{line: "10:40-12 спец", title: "спец", start: hhmm(10, 40), end: hhmm(12, 0), hasEnd: true, uncertain: true},

		// A semicolon where the colon was meant. His keyboard, his timetable.
		{line: "13;00 обед", title: "обед", start: hhmm(13, 0), uncertain: true},
		{line: "13.00 обед", title: "обед", start: hhmm(13, 0), uncertain: true},

		// A bare hour, and an hour split by a space.
		{line: "12 обед", title: "обед", start: hhmm(12, 0), uncertain: true},
		{line: "10 00 -12 физра", title: "физра", start: hhmm(10, 0), end: hhmm(12, 0), hasEnd: true, uncertain: true},
	}

	for _, c := range cases {
		p := ParseLine(c.line, tuesday15())
		if p.Title != c.title {
			t.Errorf("%q: Title = %q, want %q", c.line, p.Title, c.title)
		}
		if !p.Draft.HasTime {
			t.Errorf("%q: no time recognised", c.line)
			continue
		}
		if p.Draft.Start != c.start {
			t.Errorf("%q: Start = %v, want %v", c.line, p.Draft.Start, c.start)
		}
		if p.Draft.HasEnd != c.hasEnd {
			t.Errorf("%q: HasEnd = %v, want %v", c.line, p.Draft.HasEnd, c.hasEnd)
		}
		if c.hasEnd && p.Draft.End != c.end {
			t.Errorf("%q: End = %v, want %v", c.line, p.Draft.End, c.end)
		}
		// 🔴 Denis: «если время написано не строго по формулировке должно
		// помечать на уточнение/подтверждение». A loose reading that does not
		// say it was loose is a guess wearing the clothes of a fact.
		if got := p.Draft.IsUncertain(FieldTime); got != c.uncertain {
			t.Errorf("%q: uncertain(time) = %v, want %v", c.line, got, c.uncertain)
		}
	}
}

// 🔴 A bare number is a time only where a time is expected: at the start of the
// entry, or as the end of a range that already began with a clock. Anywhere
// else it is part of what the user wrote.
func TestBareNumberIsNotATimeInTheMiddleOfAPhrase(t *testing.T) {
	for _, line := range []string{"отжаться 12 раз", "купить 3 билета", "кабинет 12"} {
		p := ParseLine(line, tuesday15())
		if p.Draft.HasTime {
			t.Errorf("%q: read %v as a time", line, p.Draft.Start)
		}
		if p.Title != line {
			t.Errorf("%q: Title = %q, want the line unchanged", line, p.Title)
		}
	}
}

// An hour out of range is not an hour, however it is punctuated.
func TestLooseFormsStillRejectImpossibleClocks(t *testing.T) {
	for _, line := range []string{"25;00 обед", "99 обед", "10:40-99 спец"} {
		p := ParseLine(line, tuesday15())
		if p.Draft.HasTime && p.Draft.Start > 23*time.Hour+59*time.Minute {
			t.Errorf("%q: accepted %v as a time", line, p.Draft.Start)
		}
	}
}

// A day written loosely is flagged the same way a time is.
func TestLooseDateIsFlagged(t *testing.T) {
	p := ParseLine("созвон 16/09 10:00", tuesday15())
	if !p.Draft.HasDay || p.Draft.Day.Day() != 16 {
		t.Fatalf("Day = %s, want 16 September", p.Draft.Day.Format("2 Jan"))
	}
	if !p.Draft.IsUncertain(FieldDay) {
		t.Error("a date written with a slash was accepted without a warning")
	}
	if p.Title != "созвон" {
		t.Errorf("Title = %q", p.Title)
	}
}

func TestStrictFormsCarryNoWarning(t *testing.T) {
	p := ParseLine("оркестр 16.09 14:00-15:00", tuesday15())
	if p.Draft.IsUncertain(FieldTime) {
		t.Error("a strict time was flagged as loose")
	}
	if p.Draft.IsUncertain(FieldDay) {
		t.Error("a strict date was flagged as loose")
	}
}

// His whole timetable, re-read. This is the test that says the report he sent
// back is now right.
func TestHisTimetableParsesCorrectly(t *testing.T) {
	block := "Вторник:\n9:00 учу сама спец\n10:40-12 спец\n12:00 -12:50 физра\n13;00 обед\n" +
		"Среда:\n10:40-12:00 Ист музыки\n12:00 -14:40 обед\n14:50-18:00 оркестр"
	got := ParseEventList(block, tuesday15())

	want := []struct {
		title  string
		day    int
		start  time.Duration
		end    time.Duration
		hasEnd bool
	}{
		{"учу сама спец", 15, hhmm(9, 0), 0, false},
		{"спец", 15, hhmm(10, 40), hhmm(12, 0), true},
		{"физра", 15, hhmm(12, 0), hhmm(12, 50), true},
		{"обед", 15, hhmm(13, 0), 0, false},
		{"Ист музыки", 16, hhmm(10, 40), hhmm(12, 0), true},
		{"обед", 16, hhmm(12, 0), hhmm(14, 40), true},
		{"оркестр", 16, hhmm(14, 50), hhmm(18, 0), true},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Title != w.title {
			t.Errorf("entry %d: Title = %q, want %q", i, got[i].Title, w.title)
		}
		if got[i].Draft.Day.Day() != w.day {
			t.Errorf("entry %d (%s): day = %d, want %d", i, w.title, got[i].Draft.Day.Day(), w.day)
		}
		if got[i].Draft.Start != w.start {
			t.Errorf("entry %d (%s): Start = %v, want %v", i, w.title, got[i].Draft.Start, w.start)
		}
		if got[i].Draft.HasEnd != w.hasEnd || (w.hasEnd && got[i].Draft.End != w.end) {
			t.Errorf("entry %d (%s): End = %v (hasEnd %v), want %v", i, w.title, got[i].Draft.End, got[i].Draft.HasEnd, w.end)
		}
	}
}
