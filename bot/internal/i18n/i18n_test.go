package i18n

import "testing"

func TestTPicksTheLanguage(t *testing.T) {
	if got := T(RU, "Привет", "Hello"); got != "Привет" {
		t.Errorf("RU gave %q", got)
	}
	if got := T(EN, "Привет", "Hello"); got != "Hello" {
		t.Errorf("EN gave %q", got)
	}
}

// An unset or unknown language is Russian, not empty. A bot that answers with
// an empty string is broken in a way that looks like a network problem.
func TestUnknownLanguageFallsBack(t *testing.T) {
	for _, s := range []string{"", "fr", "RU", "русский", "en-GB"} {
		if got := Parse(s); got != Default && got != EN {
			t.Errorf("Parse(%q) = %q", s, got)
		}
	}
	if Parse("") != RU {
		t.Error("an unset language must be Russian")
	}
	if Parse("en") != EN {
		t.Error("«en» must be English")
	}
	if got := T(Lang("fr"), "Привет", "Hello"); got != "Привет" {
		t.Errorf("an unknown language gave %q, want the Russian", got)
	}
}

// 🔴 Each language names itself in itself. A list that reads «Русский /
// Английский» is useless to the person who needs the English one — and that is
// exactly the person reading the list.
func TestEachLanguageNamesItselfInItself(t *testing.T) {
	if Name(RU) != "Русский" {
		t.Errorf("Name(RU) = %q", Name(RU))
	}
	if Name(EN) != "English" {
		t.Errorf("Name(EN) = %q", Name(EN))
	}
}
