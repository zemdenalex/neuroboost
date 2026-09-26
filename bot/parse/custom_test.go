package parse

import (
	"testing"
	"time"
)

func tagRule(tag string) Trigger { return Trigger{Field: FieldTag, Value: tag} }

func TestCustomWordBecomesATag(t *testing.T) {
	toks := Tokenize("пробежка спорт 07:00")
	var d Draft
	recogniseTimeRange(toks, &d)
	if !RecogniseCustomTriggers(toks, map[string]Trigger{"спорт": tagRule("здоровье")}, tuesday15(), &d) {
		t.Fatal("not recognised")
	}
	if len(d.Tags) != 1 || d.Tags[0] != "здоровье" {
		t.Errorf("Tags = %v, want [здоровье]", d.Tags)
	}
	if got := Title(toks); got != "пробежка" {
		t.Errorf("Title = %q, want %q", got, "пробежка")
	}
}

// 🔴 Denis, 15.09: «Ключевые слова не обязательно теги, они должны быть как
// триггеры… выбирается что оно обозначает — тег/дата/цвет/календарь и тд».
// A shortcut that can only make tags cannot express «созвон = в рабочий
// календарь, синим», which is the whole reason to have shortcuts.
func TestCustomWordCanSetAnyCharacteristic(t *testing.T) {
	cases := []struct {
		rule  Trigger
		check func(d Draft) bool
		what  string
	}{
		{Trigger{FieldColour, "blue"}, func(d Draft) bool { return d.Colour == "blue" }, "colour"},
		{Trigger{FieldCalendar, "Работа"}, func(d Draft) bool { return d.Calendar == "Работа" }, "calendar"},
		{Trigger{FieldRepeat, "FREQ=WEEKLY"}, func(d Draft) bool { return d.Repeat == "FREQ=WEEKLY" }, "repeat"},
		{Trigger{FieldAllDay, ""}, func(d Draft) bool { return d.AllDay }, "all-day"},
		{Trigger{FieldKind, ""}, func(d Draft) bool { return d.IsTask }, "task"},
		{Trigger{FieldDay, "завтра"}, func(d Draft) bool { return d.HasDay && d.Day.Day() == 16 }, "day"},
		{Trigger{FieldTime, "14:00"}, func(d Draft) bool { return d.HasTime && d.Start == hhmm(14, 0) }, "time"},
	}
	for _, c := range cases {
		toks := Tokenize("созвон триггер")
		var d Draft
		if !RecogniseCustomTriggers(toks, map[string]Trigger{"триггер": c.rule}, tuesday15(), &d) {
			t.Errorf("%s: not recognised", c.what)
			continue
		}
		if !c.check(d) {
			t.Errorf("%s: not applied — %+v", c.what, d)
		}
		if got := Title(toks); got != "созвон" {
			t.Errorf("%s: Title = %q, want %q", c.what, got, "созвон")
		}
	}
}

// A day stored as a PHRASE keeps meaning what it says. Storing «завтра» as a
// date would freeze it to the day the word was defined.
func TestDayTriggerIsRereadEachTime(t *testing.T) {
	rules := map[string]Trigger{"планёрка": {FieldDay, "завтра"}}

	for _, c := range []struct {
		now  time.Time
		want int
	}{
		{tuesday15(), 16},
		{time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC), 21},
	} {
		toks := Tokenize("встреча планёрка")
		var d Draft
		RecogniseCustomTriggers(toks, rules, c.now, &d)
		if !d.HasDay || d.Day.Day() != c.want {
			t.Errorf("from %s: day = %s, want the %dth", c.now.Format("2 Jan"), d.Day.Format("2 Jan"), c.want)
		}
	}
}

// 🔴 A built-in meaning wins. Someone who defines «синий» as a tag must not
// lose the colour — which is why the custom pass runs last and only sees
// tokens nothing else claimed.
func TestBuiltInMeaningBeatsACustomWord(t *testing.T) {
	p := ParseLine("созвон синий", tuesday15())
	if p.Draft.Colour != "blue" {
		t.Fatalf("Colour = %q, want blue", p.Draft.Colour)
	}

	RecogniseCustomTriggers(p.Tokens, map[string]Trigger{"синий": tagRule("настроение")}, tuesday15(), &p.Draft)
	if len(p.Draft.Tags) != 0 {
		t.Errorf("the custom word overrode the colour: Tags = %v", p.Draft.Tags)
	}
	if Title(p.Tokens) != "созвон" {
		t.Errorf("Title = %q", Title(p.Tokens))
	}
}

// ⚠ A shortcut is a default, not an override. What the line says explicitly
// stands.
func TestExplicitValueBeatsATrigger(t *testing.T) {
	p := ParseLine("созвон 16.09 планёрка", tuesday15())
	RecogniseCustomTriggers(p.Tokens, map[string]Trigger{"планёрка": {FieldDay, "послезавтра"}}, tuesday15(), &p.Draft)

	if p.Draft.Day.Day() != 16 {
		t.Errorf("day = %s, want the 16th — the date in the line must win", p.Draft.Day.Format("2 Jan"))
	}
}

// A trigger whose value stopped making sense keeps its word in the title
// rather than deleting what the user wrote.
func TestBrokenTriggerLeavesTheWordAlone(t *testing.T) {
	toks := Tokenize("созвон сломанное")
	var d Draft
	RecogniseCustomTriggers(toks, map[string]Trigger{"сломанное": {FieldDay, "это не дата"}}, tuesday15(), &d)

	if d.HasDay {
		t.Error("a broken day trigger set a day anyway")
	}
	if got := Title(toks); got != "созвон сломанное" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}

func TestCustomWordsAreExactAndDeduplicated(t *testing.T) {
	toks := Tokenize("пробежка спортзал спорт спорт")
	var d Draft
	RecogniseCustomTriggers(toks, map[string]Trigger{"спорт": tagRule("здоровье")}, tuesday15(), &d)

	if len(d.Tags) != 1 {
		t.Errorf("Tags = %v, want one entry", d.Tags)
	}
	// «спортзал» merely starts with the word; it is not the word.
	if got := Title(toks); got != "пробежка спортзал" {
		t.Errorf("Title = %q, want %q", got, "пробежка спортзал")
	}
}

func TestNoCustomWordsChangesNothing(t *testing.T) {
	toks := Tokenize("пробежка спорт")
	var d Draft
	if RecogniseCustomTriggers(toks, nil, tuesday15(), &d) {
		t.Error("an empty vocabulary claimed something")
	}
	if got := Title(toks); got != "пробежка спорт" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}

// The stored names and the labels must stay in step: a name that cannot be
// resolved back is a setting that silently stops working.
func TestEveryTriggerFieldRoundTrips(t *testing.T) {
	for _, tf := range TriggerFields {
		got, ok := FieldByName(tf.Name)
		if !ok || got != tf.Field {
			t.Errorf("%q does not resolve back to its field", tf.Name)
		}
		if FieldName(tf.Field) != tf.Name {
			t.Errorf("FieldName(%v) = %q, want %q", tf.Field, FieldName(tf.Field), tf.Name)
		}
		if FieldLabel(tf.Field) != tf.Label {
			t.Errorf("FieldLabel(%v) = %q, want %q", tf.Field, FieldLabel(tf.Field), tf.Label)
		}
	}
}
