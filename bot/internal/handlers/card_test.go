package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func tuesday15() time.Time {
	return time.Date(2026, 9, 15, 19, 0, 0, 0, time.UTC)
}

func draftFrom(line string) draftState {
	p := parse.ParseLine(line, tuesday15())
	return draftState{Title: p.Title, D: p.Draft}
}

// 🔴 Denis's sentence, asked the way he asked for it: «повтор» with no
// frequency must produce a QUESTION, not a one-off event.
func TestBareRepeatIsAQuestionNotADefault(t *testing.T) {
	st := draftFrom("среда 14:00-15:00 оркестр повтор")
	if q := nextQuestion(st); q != askFreq {
		t.Errorf("nextQuestion = %q, want %q — confirming would create a one-off silently", q, askFreq)
	}

	st.D.Repeat = "FREQ=WEEKLY"
	st.D.RepeatAsked = false
	if q := nextQuestion(st); q != askNothing {
		t.Errorf("after an answer nextQuestion = %q, want nothing left", q)
	}
}

func TestNextQuestionOrder(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"среда 14:00 оркестр", askNothing},
		{"оркестр", askDate},
		{"среда оркестр", askTime},
		{"анализы весь день четверг", askNothing},
		{"среда 14:00 повтор", askTitle}, // nothing left for a title
		{"среда 14:00 оркестр повтор", askFreq},
	}
	for _, c := range cases {
		if got := nextQuestion(draftFrom(c.line)); got != c.want {
			t.Errorf("%q: nextQuestion = %q, want %q", c.line, got, c.want)
		}
	}
}

// 🔴 What is missing is printed, not omitted. A card that silently leaves the
// date line out reads as "no date needed" — and "absent looks the same as
// empty" is a defect this product has already shipped once.
func TestCardNamesWhatIsMissing(t *testing.T) {
	card := renderDraft(draftFrom("оркестр"), tuesday15())
	if !strings.Contains(card, "дата не указана") {
		t.Errorf("no missing-date line in:\n%s", card)
	}
	if !strings.Contains(card, "время не указано") {
		t.Errorf("no missing-time line in:\n%s", card)
	}
}

func TestCardShowsWhatWasUnderstood(t *testing.T) {
	st := draftFrom("среда 14:00-15:00 оркестр повтор синий #музыка")
	st.CalendarName = "Работа"
	card := renderDraft(st, tuesday15())

	for _, want := range []string{
		"оркестр",
		"среда, 16 сентября",
		"14:00–15:00",
		"частота не указана",
		"синий",
		"Работа",
		"музыка",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("card does not mention %q:\n%s", want, card)
		}
	}
}

// A repeating range that crosses midnight prints two clock times, not a
// negative one: 23:00–01:00 is two hours.
func TestCardPrintsAMidnightCrossing(t *testing.T) {
	card := renderDraft(draftFrom("смена 23:00-01:00 завтра"), tuesday15())
	if !strings.Contains(card, "23:00–01:00") {
		t.Errorf("card does not print the crossing range:\n%s", card)
	}
}

func TestCardMarksATask(t *testing.T) {
	card := renderDraft(draftFrom("отжаться задача завтра 10:00"), tuesday15())
	if !strings.Contains(card, "✅ <b>отжаться</b>") {
		t.Errorf("a task is not marked as one:\n%s", card)
	}
}

// 🔴 Denis, 15.09: «должно быть дата потом время потом название, чтобы было
// проще сориентироваться». The order is the requirement, so the order is what
// is asserted — not merely that all three appear.
func TestCardPutsDateThenTimeThenTitle(t *testing.T) {
	card := renderDraft(draftFrom("оркестр среда 14:00-15:00"), tuesday15())

	day := strings.Index(card, "🗓")
	clock := strings.Index(card, "🕐")
	title := strings.Index(card, "оркестр")

	if day < 0 || clock < 0 || title < 0 {
		t.Fatalf("a line is missing entirely:\n%s", card)
	}
	if !(day < clock && clock < title) {
		t.Errorf("order is date %d, time %d, title %d — want date, then time, then title:\n%s",
			day, clock, title, card)
	}
}

// 🔴 A loose reading has to announce itself. «13;00» is read as 13:00 now,
// which is what he asked for; reading it without the mark would turn a typo
// into a fact he never checked.
func TestCardFlagsALooseReading(t *testing.T) {
	loose := renderDraft(draftFrom("обед завтра 13;00"), tuesday15())
	if !strings.Contains(loose, "⚠ проверь") {
		t.Errorf("«13;00» was read silently:\n%s", loose)
	}

	// The contrast that makes the assertion mean something: a strict time
	// carries no mark, or every line would carry one and the mark would say
	// nothing.
	strict := renderDraft(draftFrom("обед завтра 13:00"), tuesday15())
	if strings.Contains(strict, "⚠ проверь") {
		t.Errorf("a strict time was flagged too:\n%s", strict)
	}
}

// 🔴 Denis, 15.09: «даже если тег это полдень то это тег, а уже не время».
// The tag branch does not parse; it splits.
func TestTagEditingHonoursNoKeyword(t *testing.T) {
	tags := splitTags("полдень, среда, весь день, Музыка, музыка")
	want := []string{"полдень", "среда", "весь день", "музыка"}
	if len(tags) != len(want) {
		t.Fatalf("got %v, want %v", tags, want)
	}
	for i := range want {
		if tags[i] != want[i] {
			t.Errorf("tag %d = %q, want %q", i, tags[i], want[i])
		}
	}
}

func TestTagEditingIgnoresEmptyPieces(t *testing.T) {
	if tags := splitTags(" , ,, #дела ,"); len(tags) != 1 || tags[0] != "дела" {
		t.Errorf("got %v, want [дела]", tags)
	}
}
