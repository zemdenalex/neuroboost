package parse

import (
	"testing"
	"time"
)

// 🔴 Denis's event block, item 5, verbatim. Today is Tuesday 15.09.2026, so
// «среда» is the 16th and «четверг» the 17th — «и все это на этой неделе,
// потому что нет обозначения».
func TestDenisEventBlock(t *testing.T) {
	block := "среда\n10:00 завтрак\n12:00-14:00 оркестр\n17:00 работа\nчетверг\n8:00 анализы"
	got := ParseEventList(block, tuesday15())

	want := []struct {
		title string
		day   int
		start time.Duration
	}{
		{"завтрак", 16, hhmm(10, 0)},
		{"оркестр", 16, hhmm(12, 0)},
		{"работа", 16, hhmm(17, 0)},
		{"анализы", 17, hhmm(8, 0)},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Title != w.title {
			t.Errorf("entry %d: Title = %q, want %q", i, got[i].Title, w.title)
		}
		if !got[i].Draft.HasDay || got[i].Draft.Day.Day() != w.day {
			t.Errorf("entry %d (%s): day = %s, want %d September",
				i, w.title, got[i].Draft.Day.Format("2 Jan"), w.day)
		}
		if got[i].Draft.Start != w.start {
			t.Errorf("entry %d (%s): Start = %v, want %v", i, w.title, got[i].Draft.Start, w.start)
		}
	}
	// The range on the second line survives as a range.
	if !got[1].Draft.HasEnd || got[1].Draft.End != hhmm(14, 0) {
		t.Errorf("оркестр: end = %v (hasEnd %v), want 14:00", got[1].Draft.End, got[1].Draft.HasEnd)
	}
}

// 🔴 A day line with words after it is an event, not a header. Treating it as a
// header would silently drop a title the user typed — the same class of loss as
// the substring bug.
func TestDayLineWithATitleIsAnEventNotAHeader(t *testing.T) {
	got := ParseEventList("четверг выходной\n10:00 завтрак", tuesday15())
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Title != "выходной" {
		t.Errorf("entry 0: Title = %q, want %q", got[0].Title, "выходной")
	}
	// It still moves the day for what follows.
	if got[1].Draft.Day.Day() != 17 {
		t.Errorf("завтрак landed on %s, want 17 September", got[1].Draft.Day.Format("2 Jan"))
	}
}

// «завтра … послезавтра … пятница …» — his words: three different days.
func TestRelativeDayHeadersAlsoInherit(t *testing.T) {
	got := ParseEventList("завтра\n10:00 а\nпослезавтра\n11:00 б\nпятница\n12:00 в", tuesday15())
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}
	for i, want := range []int{16, 17, 18} {
		if got[i].Draft.Day.Day() != want {
			t.Errorf("entry %d landed on %s, want %d September", i, got[i].Draft.Day.Format("2 Jan"), want)
		}
	}
}

// Lines before any header carry no day at all — the card asks rather than
// guessing that they mean today.
func TestLinesBeforeAnyHeaderHaveNoDay(t *testing.T) {
	got := ParseEventList("10:00 завтрак\nсреда\n11:00 оркестр", tuesday15())
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].Draft.HasDay {
		t.Errorf("завтрак was given the day %s out of nowhere", got[0].Draft.Day.Format("2 Jan"))
	}
	if got[1].Draft.Day.Day() != 16 {
		t.Errorf("оркестр landed on %s, want 16", got[1].Draft.Day.Format("2 Jan"))
	}
}

func TestTaskListStripsMarkers(t *testing.T) {
	for _, block := range []string{
		"1. Отжаться\n2. Подтянуться\n3. Присесть",
		"- Отжаться\n- Подтянуться\n- Присесть",
		"Отжаться\nПодтянуться\nПрисесть",
		"Отжаться, Подтянуться, Присесть",
	} {
		got := ParseTaskList(block, tuesday15())
		if len(got) != 3 {
			t.Errorf("%q: got %d tasks, want 3", block, len(got))
			continue
		}
		if got[0].Title != "Отжаться" || got[2].Title != "Присесть" {
			t.Errorf("%q: titles = %q, %q, %q", block, got[0].Title, got[1].Title, got[2].Title)
		}
	}
}

// ⚠ «16.09» and «1.5 часа» begin with a digit and a dot and are NOT markers.
// A marker needs the space.
func TestStripMarkerLeavesDatesAndQuantitiesAlone(t *testing.T) {
	for _, line := range []string{"16.09 оркестр", "1.5 часа работы", "2)нет пробела"} {
		if got := StripMarker(line); got != line {
			t.Errorf("StripMarker(%q) = %q, want unchanged", line, got)
		}
	}
	if got := StripMarker("1. Отжаться"); got != "Отжаться" {
		t.Errorf("StripMarker(%q) = %q", "1. Отжаться", got)
	}
}

// 🔴 The bot asks; it never decides. Denis: «это все должно уточняться».
func TestLooksLikeList(t *testing.T) {
	cases := map[string]bool{
		"1. Отжаться\n2. Подтянуться": true,
		"Отжаться\nПодтянуться":       true,
		"- Отжаться":                  true,
		"Отжаться, подтянуться":       true,
		"Ужин, завтра 19:00":          false, // a comma next to a time is punctuation
		"Позвонить Ивану":             false,
		"среда 14:00 оркестр":         false,
	}
	for text, want := range cases {
		if got := LooksLikeList(text, tuesday15()); got != want {
			t.Errorf("LooksLikeList(%q) = %v, want %v", text, got, want)
		}
	}
}

// One line with commas is split only after the user says it is a list; the
// splitting itself must then be right.
func TestEntriesSplitsASingleLineOnCommasOnly(t *testing.T) {
	if got := Entries("а, б, в"); len(got) != 3 {
		t.Errorf("got %v, want 3 entries", got)
	}
	// Two lines: the comma inside one of them is punctuation, not a separator.
	if got := Entries("а, б\nв"); len(got) != 2 {
		t.Errorf("got %v, want 2 entries", got)
	}
}

// 🔴 A task list reads the TASK vocabulary, not the event one. Denis, 16.09:
// «Отжаться 1ч» came back as a task literally called «Отжаться 1ч», while the
// same words typed as a single task parsed correctly — because the single path
// used ParseTask and the list path used ParseLine.
func TestTaskListReadsEstimatesAndPriorities(t *testing.T) {
	block := "1. Отжаться 1ч\n2. Подтянуться 10м !1\n3. Присесть 5 мин #спорт"
	got := ParseTaskList(block, tuesday15())

	if len(got) != 3 {
		t.Fatalf("got %d tasks, want 3", len(got))
	}
	if got[0].Title != "Отжаться" {
		t.Errorf("task 0 title = %q, want %q — the estimate stayed in the name", got[0].Title, "Отжаться")
	}
	if got[0].EstimatedMinutes == nil || *got[0].EstimatedMinutes != 60 {
		t.Errorf("task 0 estimate = %v, want 60", got[0].EstimatedMinutes)
	}
	if got[1].Title != "Подтянуться" {
		t.Errorf("task 1 title = %q", got[1].Title)
	}
	if got[1].Priority == nil || *got[1].Priority != 1 {
		t.Errorf("task 1 priority = %v, want 1", got[1].Priority)
	}
	if got[2].Title != "Присесть" {
		t.Errorf("task 2 title = %q", got[2].Title)
	}
	if len(got[2].Tags) != 1 || got[2].Tags[0] != "спорт" {
		t.Errorf("task 2 tags = %v", got[2].Tags)
	}
}
