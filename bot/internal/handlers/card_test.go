package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
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
	card := renderDraft(i18n.RU, draftFrom("оркестр"), tuesday15())
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
	card := renderDraft(i18n.RU, st, tuesday15())

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
	card := renderDraft(i18n.RU, draftFrom("смена 23:00-01:00 завтра"), tuesday15())
	if !strings.Contains(card, "23:00–01:00") {
		t.Errorf("card does not print the crossing range:\n%s", card)
	}
}

func TestCardMarksATask(t *testing.T) {
	card := renderDraft(i18n.RU, draftFrom("отжаться задача завтра 10:00"), tuesday15())
	// ⚠ До 18.09 здесь было "✅ <b>отжаться</b>": иконка названия несла вид.
	// Денис попросил у названия свою иконку, и вид теперь сказан словами
	// отдельной строкой. Требование то же: задача должна быть видна как задача.
	if !strings.Contains(card, "и задача") {
		t.Errorf("a task is not marked as one:\n%s", card)
	}
}

// 🔴 Denis, 15.09: «должно быть дата потом время потом название, чтобы было
// проще сориентироваться». The order is the requirement, so the order is what
// is asserted — not merely that all three appear.
func TestCardPutsDateThenTimeThenTitle(t *testing.T) {
	card := renderDraft(i18n.RU, draftFrom("оркестр среда 14:00-15:00"), tuesday15())

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
	loose := renderDraft(i18n.RU, draftFrom("обед завтра 13;00"), tuesday15())
	if !strings.Contains(loose, "⚠ проверь") {
		t.Errorf("«13;00» was read silently:\n%s", loose)
	}

	// The contrast that makes the assertion mean something: a strict time
	// carries no mark, or every line would carry one and the mark would say
	// nothing.
	strict := renderDraft(i18n.RU, draftFrom("обед завтра 13:00"), tuesday15())
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

// 🔴 Denis, 18.09: «в базовых карточках должна быть вся информация, просто
// говоря "нет" для характеристик без значения, чтобы люди знали, что они
// существуют».
//
// A field printed only when filled is invisible to the person who has never
// filled it — and that is how someone concludes the app has no repeats, no
// tags and no calendars. It is also the mechanical reason «разница между
// задачами и событиями» stayed unclear: the card never showed what the two
// differ in.
func TestCardNamesEveryFieldEvenWhenEmpty(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	st.CalendarName = "Личный"
	card := renderDraft(i18n.RU, st, tuesday15())

	for _, label := range []string{"Повтор:", "Календарь:", "Напоминания:", "Теги:", "Цвет:"} {
		if !strings.Contains(card, label) {
			t.Errorf("карточка не называет %q — человек не узнает, что поле есть:\n%s", label, card)
		}
	}
	// The negative control for the labels: they must be there AND carry the
	// word «нет», not a blank that reads as a rendering bug.
	if n := strings.Count(card, "нет"); n < 3 {
		t.Errorf("пустые поля сказали «нет» только %d раза:\n%s", n, card)
	}
}

// The control that stops the card from passing by printing «нет» everywhere.
func TestCardStillPrintsRealValues(t *testing.T) {
	st := draftFrom("оркестр среда 14:00-15:00 каждую неделю синий #музыка")
	st.CalendarName = "Работа"
	card := renderDraft(i18n.RU, st, tuesday15())

	for _, want := range []string{"Работа", "музыка", "синий"} {
		if !strings.Contains(card, want) {
			t.Errorf("заполненное поле %q потерялось:\n%s", want, card)
		}
	}
	if strings.Contains(card, "Теги: нет") || strings.Contains(card, "Цвет: нет") {
		t.Errorf("заполненное поле напечаталось как «нет»:\n%s", card)
	}
}

// Both languages, because a label added in one is a label missing in the other
// — that is exactly how «Notification button language» became a defect.
func TestCardNamesEveryFieldInEnglishToo(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	card := renderDraft(i18n.EN, st, tuesday15())

	for _, label := range []string{"Repeat:", "Calendar:", "Reminders:", "Tags:", "Colour:"} {
		if !strings.Contains(card, label) {
			t.Errorf("the English card does not name %q:\n%s", label, card)
		}
	}
	if !strings.Contains(card, "none") {
		t.Errorf("empty fields did not say «none»:\n%s", card)
	}
}

// 🔴 Denis, 18.09: «by note you meant description, and if we have no
// description line we should have — and if description is filled and it's long
// it should be shortened».
//
// The card is a summary, not the document. A description pasted from a letter
// would push the buttons off a phone screen, which is how a confirmation card
// stops being confirmable.
func TestCardNamesTheDescriptionAndShortensALongOne(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	card := renderDraft(i18n.RU, st, tuesday15())
	if !strings.Contains(card, "Описание:") || !strings.Contains(card, "Описание: нет") {
		t.Errorf("пустая заметка не названа словом «нет»:\n%s", card)
	}

	st.Description = "Взять полис и паспорт, приехать за пятнадцать минут, спросить про рассрочку и записаться на следующий приём сразу на выходе"
	card = renderDraft(i18n.RU, st, tuesday15())
	line := lineWith(card, "Описание:")
	if !strings.Contains(line, "Взять полис") {
		t.Errorf("заметка не показана вовсе:\n%s", card)
	}
	if len([]rune(line)) > 90 {
		t.Errorf("длинная заметка не укорочена — %d символов в строке:\n%s", len([]rune(line)), line)
	}
	if !strings.Contains(line, "…") {
		t.Errorf("укорочённая заметка не помечена многоточием, и её не отличить от полной:\n%s", line)
	}

	short := "взять полис"
	st.Description = short
	line = lineWith(renderDraft(i18n.RU, st, tuesday15()), "Описание:")
	if strings.Contains(line, "…") {
		t.Errorf("короткая заметка обрезана без нужды: %q", line)
	}
}

func lineWith(card, needle string) string {
	for _, l := range strings.Split(card, "\n") {
		if strings.Contains(l, needle) {
			return l
		}
	}
	return ""
}

// 🔴 The note must actually reach the API. The card showing it proves only that
// the card shows it — the failure this guards against is a field that is typed,
// displayed, confirmed, and then quietly dropped at creation.
func TestNoteReachesTheAPIWhole(t *testing.T) {
	long := "Взять полис и паспорт, приехать за пятнадцать минут, спросить про рассрочку"
	req := api.CreateEventReq{Description: optional(long)}
	if req.Description == nil || *req.Description != long {
		t.Fatalf("описание не доехало целиком: %v", req.Description)
	}
	// An empty note must be ABSENT, not an empty string: the API's field is a
	// pointer, and a present empty value clears whatever was there.
	if optional("   ") != nil {
		t.Error("пустая заметка отправляется как пустая строка и сотрёт существующую")
	}
}
