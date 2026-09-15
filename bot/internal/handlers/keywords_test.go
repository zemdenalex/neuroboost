package handlers

import "testing"

func TestSplitKeyword(t *testing.T) {
	cases := []struct {
		text, word, tag string
	}{
		{"спорт = здоровье", "спорт", "здоровье"},
		{"спорт=здоровье", "спорт", "здоровье"},
		{"Спорт → Здоровье", "спорт", "здоровье"},
		{"спорт -> здоровье", "спорт", "здоровье"},
		{"спорт: здоровье", "спорт", "здоровье"},
		// One word tags itself.
		{"спорт", "спорт", "спорт"},
		{"  СПОРТ  ", "спорт", "спорт"},
	}
	for _, c := range cases {
		word, tag := splitKeyword(c.text)
		if word != c.word || tag != c.tag {
			t.Errorf("splitKeyword(%q) = %q, %q; want %q, %q", c.text, word, tag, c.word, c.tag)
		}
	}
}

// 🔴 Two words with no separator are ambiguous — «утренняя пробежка» could be
// one word-pair or a word and a tag — and ambiguity is refused rather than
// guessed. A wrong guess here silently changes how every future line parses.
func TestSplitKeywordRefusesTheAmbiguous(t *testing.T) {
	for _, text := range []string{"", "   ", "утренняя пробежка", "= здоровье", "спорт =", "a b c"} {
		if word, tag := splitKeyword(text); word != "" || tag != "" {
			t.Errorf("splitKeyword(%q) = %q, %q; want a refusal", text, word, tag)
		}
	}
}

// Only the first separator splits: a tag may contain one, a word may not.
func TestSplitKeywordSplitsOnlyOnce(t *testing.T) {
	word, tag := splitKeyword("спорт = здоровье = важно")
	if word != "спорт" || tag != "здоровье = важно" {
		t.Errorf("got %q, %q", word, tag)
	}
}
