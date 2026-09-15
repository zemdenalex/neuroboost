package parse

import "testing"

func TestCustomWordBecomesATag(t *testing.T) {
	toks := Tokenize("пробежка спорт 07:00")
	var d Draft
	recogniseTimeRange(toks, &d)
	if !RecogniseCustomTags(toks, map[string]string{"спорт": "здоровье"}, &d) {
		t.Fatal("not recognised")
	}
	if len(d.Tags) != 1 || d.Tags[0] != "здоровье" {
		t.Errorf("Tags = %v, want [здоровье]", d.Tags)
	}
	if got := Title(toks); got != "пробежка" {
		t.Errorf("Title = %q, want %q", got, "пробежка")
	}
}

// 🔴 A built-in meaning wins. Someone who defines «синий» as a tag must not
// lose the colour — which is why the custom pass runs last and only sees
// tokens nothing else claimed.
func TestBuiltInMeaningBeatsACustomWord(t *testing.T) {
	line := "созвон синий"
	p := ParseLine(line, tuesday15())
	if p.Draft.Colour != "blue" {
		t.Fatalf("Colour = %q, want blue", p.Draft.Colour)
	}

	RecogniseCustomTags(p.Tokens, map[string]string{"синий": "настроение"}, &p.Draft)
	if len(p.Draft.Tags) != 0 {
		t.Errorf("the custom word overrode the colour: Tags = %v", p.Draft.Tags)
	}
	if Title(p.Tokens) != "созвон" {
		t.Errorf("Title = %q", Title(p.Tokens))
	}
}

func TestCustomWordsAreExactAndDeduplicated(t *testing.T) {
	toks := Tokenize("пробежка спортзал спорт спорт")
	var d Draft
	RecogniseCustomTags(toks, map[string]string{"спорт": "здоровье"}, &d)

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
	if RecogniseCustomTags(toks, nil, &d) {
		t.Error("an empty vocabulary claimed something")
	}
	if got := Title(toks); got != "пробежка спорт" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}
