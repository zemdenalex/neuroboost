package parse

import (
	"strings"
	"testing"
	"time"
)

// Denis's line of 21.09, verbatim, and what it became.
//
// 🔴 It became ONE task, titled «пить таблетки, полить цветы через день, отчёт
// каждую неделю, раз дня протереть пыль» — note «раз ДНЯ»: «в 3» had been
// eaten by the time recogniser out of the middle of the fourth clause. And
// that same misread is what silenced the question: the old heuristic asked
// «does this line have a time?» and, finding 03:00, concluded the commas were
// punctuation.
//
// Two defects holding each other up, which is why this test asserts the
// question rather than the parse: whatever the recognisers do inside one lump,
// a line of four chores must be handed back to the user as a question.
const fourChores = "ежедневно пить таблетки, полить цветы через день, отчёт каждую неделю, раз в 3 дня протереть пыль"

func refDay() time.Time {
	return time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
}

func TestFourChoresInOneLineAreAQuestion(t *testing.T) {
	if !LooksLikeList(fourChores, refDay()) {
		t.Error("four clauses, each a chore, went through as one task without asking")
	}

	entries := Entries(fourChores)
	if len(entries) != 4 {
		t.Fatalf("Entries gave %d parts, want 4: %q", len(entries), entries)
	}

	// 🔴 And each part must be read ON ITS OWN. Parsed together, the fourth
	// clause's «в 3» leaked into the first one's title as a 03:00 start.
	last := ParseTask(entries[3], refDay())
	if !strings.Contains(last.Title, "протереть пыль") {
		t.Errorf("last chore's title = %q, want it to keep «протереть пыль»", last.Title)
	}
	if last.Rrule == "" {
		t.Errorf("«раз в 3 дня» lost its repeat when read on its own: %+v", last)
	}
}

// 🔴 The case the old rule existed to protect. It must still hold, or fixing
// Denis's line would start interrogating every sentence with a comma in it.
func TestATimeAfterACommaIsStillPunctuation(t *testing.T) {
	for _, line := range []string{
		"Ужин, завтра 19:00",
		"Обед, 14:00",
		"Оркестр, 14:50",
	} {
		if LooksLikeList(line, refDay()) {
			t.Errorf("%q was taken for a list; the part after the comma is its time, not an item", line)
		}
	}
}

// A part that is only a date or only a time is not an item — that is the whole
// distinction this rests on, so it is asserted directly.
func TestOnlyPartsWithATitleCount(t *testing.T) {
	if n := commaItems("Ужин, завтра 19:00", refDay()); n != 1 {
		t.Errorf("commaItems = %d, want 1 — «завтра 19:00» leaves no title", n)
	}
	if n := commaItems(fourChores, refDay()); n != 4 {
		t.Errorf("commaItems = %d, want 4", n)
	}
}
