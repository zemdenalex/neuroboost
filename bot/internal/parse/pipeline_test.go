package parse

import (
	"strings"
	"testing"
	"time"
)

// 🔴 Denis's sentence from item 1, end to end. Today is Tuesday 15.09.2026.
//
// «если я пишу среда 14:00-15:00 оркестр повтор, то бот меня должен понять как
// создать событие в среду 16.09 с 14:00 до 15:00 с названием оркестр, и
// уточнить, поскольу повтор я не написал частоту»
func TestDenisSentence(t *testing.T) {
	p := ParseLine("среда 14:00-15:00 оркестр повтор", tuesday15())

	if p.Title != "оркестр" {
		t.Errorf("Title = %q, want %q", p.Title, "оркестр")
	}
	if !p.Draft.HasDay || p.Draft.Day.Day() != 16 || p.Draft.Day.Month() != time.September {
		t.Errorf("Day = %s, want 16 September", p.Draft.Day.Format("2 January"))
	}
	if p.Draft.Start != hhmm(14, 0) || p.Draft.End != hhmm(15, 0) || !p.Draft.HasEnd {
		t.Errorf("time = %v–%v (hasEnd %v), want 14:00–15:00", p.Draft.Start, p.Draft.End, p.Draft.HasEnd)
	}
	if !p.Draft.RepeatAsked {
		t.Error("RepeatAsked is false — the bot would create a one-off without asking")
	}
	if p.Draft.Repeat != "" {
		t.Errorf("Repeat = %q, want empty until he answers", p.Draft.Repeat)
	}
}

func TestParseLineTable(t *testing.T) {
	cases := []struct {
		line   string
		title  string
		day    int
		start  time.Duration
		allDay bool
		colour string
		repeat string
		isTask bool
	}{
		{line: "Ужин завтра 19:00", title: "Ужин", day: 16, start: hhmm(19, 0)},
		{line: "10:00 завтрак", title: "завтрак", day: 0, start: hhmm(10, 0)},
		{line: "анализы весь день четверг", title: "анализы", day: 17, allDay: true},
		{line: "созвон синий 16.09 10:00", title: "созвон", day: 16, start: hhmm(10, 0), colour: "blue"},
		{line: "отжаться задача", title: "отжаться", isTask: true},
		{line: "зарядка каждый день 07:00", title: "зарядка", start: hhmm(7, 0), repeat: "FREQ=DAILY"},
		{line: "обед полдень", title: "обед", start: hhmm(12, 0)},
	}

	for _, c := range cases {
		p := ParseLine(c.line, tuesday15())
		if p.Title != c.title {
			t.Errorf("%q: Title = %q, want %q", c.line, p.Title, c.title)
		}
		if c.day != 0 && (!p.Draft.HasDay || p.Draft.Day.Day() != c.day) {
			t.Errorf("%q: Day = %s, want %d", c.line, p.Draft.Day.Format("2 Jan"), c.day)
		}
		if c.day == 0 && p.Draft.HasDay {
			t.Errorf("%q: a day was invented: %s", c.line, p.Draft.Day.Format("2 Jan"))
		}
		if !c.allDay && c.start != 0 && p.Draft.Start != c.start {
			t.Errorf("%q: Start = %v, want %v", c.line, p.Draft.Start, c.start)
		}
		if p.Draft.AllDay != c.allDay {
			t.Errorf("%q: AllDay = %v, want %v", c.line, p.Draft.AllDay, c.allDay)
		}
		if p.Draft.Colour != c.colour {
			t.Errorf("%q: Colour = %q, want %q", c.line, p.Draft.Colour, c.colour)
		}
		if p.Draft.Repeat != c.repeat {
			t.Errorf("%q: Repeat = %q, want %q", c.line, p.Draft.Repeat, c.repeat)
		}
		if p.Draft.IsTask != c.isTask {
			t.Errorf("%q: IsTask = %v, want %v", c.line, p.Draft.IsTask, c.isTask)
		}
	}
}

// 🔴 The order of the recognisers is a decision, stated here literally so a
// refactor that reorders them fails instead of changing behaviour quietly.
// Three adjacencies in it are load-bearing; the comment on `recognisers` says
// which and why.
func TestRecogniserOrderIsFixed(t *testing.T) {
	want := []string{
		"date", "relative-day", "repeat", "weekday",
		"time-range", "time-word", "all-day", "kind", "colour", "tags",
	}
	if len(recognisers) != len(want) {
		t.Fatalf("got %d recognisers, want %d — if one was added, decide where it "+
			"belongs in the order and say so here", len(recognisers), len(want))
	}
	for i, w := range want {
		if recognisers[i].Name != w {
			t.Errorf("recogniser %d is %q, want %q", i, recognisers[i].Name, w)
		}
	}
}

// 🔴 Denis, 15.09: «если выбирается изменить название то там уже слова не
// учитываются а текст напрямую в название переходит». Every keyword in the
// line survives as ordinary words.
func TestRawTitleHonoursNoKeyword(t *testing.T) {
	line := "среда весь день повтор синий задача полдень"
	if got := ParseLineRaw(line); got != line {
		t.Errorf("ParseLineRaw = %q, want the line unchanged", got)
	}

	// The contrast that makes the previous assertion mean something: the same
	// line through the normal path keeps nothing at all.
	if p := ParseLine(line, tuesday15()); p.Title != "" {
		t.Errorf("ParseLine kept %q — then the raw path is not distinguishable", p.Title)
	}
}

// Tokens survive the parse. This is what lets the card show what it consumed
// and lets one field be replaced without re-running the others.
func TestParsedKeepsItsTokens(t *testing.T) {
	p := ParseLine("среда оркестр", tuesday15())
	if len(p.Tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(p.Tokens))
	}
	if p.Tokens[0].Field != FieldDay {
		t.Errorf("token %q was not marked as the day", p.Tokens[0].Text)
	}
	if p.Tokens[1].Field != FieldNone {
		t.Errorf("token %q was claimed by %v, want nothing", p.Tokens[1].Text, p.Tokens[1].Field)
	}
}

// A line that is only a title stays a line that is only a title. Nothing is
// invented — no day, no time — because the card has to be able to say "when?"
// rather than guess.
func TestBareTitleInventsNothing(t *testing.T) {
	p := ParseLine("Позвонить Ивану", tuesday15())
	if p.Title != "Позвонить Ивану" {
		t.Errorf("Title = %q", p.Title)
	}
	if p.Draft.HasDay || p.Draft.HasTime || p.Draft.AllDay ||
		p.Draft.Repeat != "" || p.Draft.RepeatAsked || p.Draft.Colour != "" || p.Draft.IsTask {
		t.Errorf("something was invented: %+v", p.Draft)
	}
}

// The guide the bot prints and what the parser does must agree. The examples
// live here as a table; §9 of the spec is what they are copied from, and
// handlers/events.go prints them.
func TestGuideExamplesParseAsAdvertised(t *testing.T) {
	cases := []struct{ line, title, promise string }{
		{"Ужин завтра 19:00", "Ужин", "завтра, 19:00"},
		{"среда 14:00-15:00 оркестр повтор", "оркестр", "ср, 14:00–15:00, спрошу частоту"},
		{"анализы весь день четверг", "анализы", "чт, весь день"},
	}
	for _, c := range cases {
		p := ParseLine(c.line, tuesday15())
		if p.Title != c.title {
			t.Errorf("guide example %q promises the title %q, parses to %q (%s)",
				c.line, c.title, p.Title, c.promise)
		}
		if !p.Draft.HasDay {
			t.Errorf("guide example %q promises a day and none was parsed", c.line)
		}
		if !p.Draft.HasTime && !p.Draft.AllDay {
			t.Errorf("guide example %q promises a time and none was parsed", c.line)
		}
	}
}

// Nothing a recogniser claims may reappear in the title. This is the invariant
// the whole package exists for, checked against every keyword at once rather
// than one test per word.
func TestNoKeywordEverReachesTheTitle(t *testing.T) {
	keywords := []string{
		"завтра", "послезавтра", "сегодня", "среда", "в среду", "следующая среда",
		"пн", "wednesday", "wed", "14:00", "14:00-15:00", "полдень", "утром",
		"весь день", "повтор", "еженедельно", "каждый день", "задача",
		"синий", "фиолетовая", "#дела", "16.09",
	}
	for _, k := range keywords {
		p := ParseLine("оркестр "+k, tuesday15())
		if p.Title != "оркестр" {
			t.Errorf("%q left %q in the title", k, strings.TrimPrefix(p.Title, "оркестр "))
		}
	}
}
