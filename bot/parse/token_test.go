package parse

import "testing"

// 🔴 The bug that started this work, in Denis's own example data.
//
// parse.Parse matched relative-day words with strings.Index, so "завтра" was
// found INSIDE "завтрак": the title lost six letters and the event moved a day.
// Tokenising by word makes the boundary free — a word either is the keyword or
// is not.
func TestTokenNormNeverMatchesInsideAnotherWord(t *testing.T) {
	cases := []struct {
		line  string
		want  []string // expected Norm values, in order
		notIn string   // a keyword that must NOT equal any token
	}{
		{"10:00 завтрак", []string{"10:00", "завтрак"}, "завтра"},
		{"pay the money monday", []string{"pay", "the", "money", "monday"}, "mon"},
		{"ужин втроём", []string{"ужин", "втроём"}, "вт"},
		{"sunny walk", []string{"sunny", "walk"}, "sun"},
	}
	for _, c := range cases {
		toks := Tokenize(c.line)
		if len(toks) != len(c.want) {
			t.Errorf("%q: got %d tokens, want %d", c.line, len(toks), len(c.want))
			continue
		}
		for i, w := range c.want {
			if toks[i].Norm != w {
				t.Errorf("%q: token %d Norm = %q, want %q", c.line, i, toks[i].Norm, w)
			}
			if toks[i].Norm == c.notIn {
				t.Errorf("%q: token %d equals the keyword %q", c.line, i, c.notIn)
			}
		}
	}
}

// Norm is what gets matched; Text is what rebuilds the title. Keeping them
// apart is what lets «Оркестр,» match a keyword list and still print its comma.
func TestNormStripsPunctuationAndCaseButTextKeepsIt(t *testing.T) {
	toks := Tokenize("Оркестр, ЗАВТРА!")
	if len(toks) != 2 {
		t.Fatalf("got %d tokens, want 2", len(toks))
	}
	if toks[0].Text != "Оркестр," {
		t.Errorf("Text = %q, want %q", toks[0].Text, "Оркестр,")
	}
	if toks[0].Norm != "оркестр" {
		t.Errorf("Norm = %q, want %q", toks[0].Norm, "оркестр")
	}
	if toks[1].Norm != "завтра" {
		t.Errorf("Norm = %q, want %q", toks[1].Norm, "завтра")
	}
}

// A time token must survive normalisation intact: trimming punctuation from
// the edges must not eat the colon or the dash of a range.
func TestNormKeepsTimeShapes(t *testing.T) {
	for _, want := range []string{"14:00", "14:00-15:00", "16.09", "14.00"} {
		toks := Tokenize(want)
		if len(toks) != 1 {
			t.Fatalf("%q: got %d tokens, want 1", want, len(toks))
		}
		if toks[0].Norm != want {
			t.Errorf("Norm = %q, want %q", toks[0].Norm, want)
		}
	}
}

// Title is the join of what nothing claimed. This is what replaces the old
// cut-the-substring-out-of-the-string approach, and it is why a recognised
// keyword can never leave a fragment behind.
func TestTitleIsWhatNothingClaimed(t *testing.T) {
	toks := Tokenize("среда 14:00-15:00 оркестр повтор")
	toks[0].Field = FieldDay
	toks[1].Field = FieldTime
	toks[3].Field = FieldRepeat

	if got := Title(toks); got != "оркестр" {
		t.Errorf("Title = %q, want %q", got, "оркестр")
	}
}

func TestTitleKeepsOriginalCasingAndInnerPunctuation(t *testing.T) {
	toks := Tokenize("Созвон с Иваном, срочно завтра")
	toks[4].Field = FieldDay

	if got := Title(toks); got != "Созвон с Иваном, срочно" {
		t.Errorf("Title = %q, want %q", got, "Созвон с Иваном, срочно")
	}
}

// Trailing separators left behind by a claimed tail are trimmed, the way the
// old cleanTitle did — otherwise "Ужин, завтра" would end in a comma.
func TestTitleTrimsDanglingSeparators(t *testing.T) {
	toks := Tokenize("Ужин, завтра")
	toks[1].Field = FieldDay

	if got := Title(toks); got != "Ужин" {
		t.Errorf("Title = %q, want %q", got, "Ужин")
	}
}

func TestTokenizeEmptyLine(t *testing.T) {
	if toks := Tokenize("   "); len(toks) != 0 {
		t.Errorf("got %d tokens, want 0", len(toks))
	}
}
