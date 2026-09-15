package parse

import "strings"

// RecogniseCustomTags claims words the user defined themselves.
//
// 🔴 It runs LAST, after every built-in recogniser, and that order is the
// answer to an obvious collision: a custom word may be spelled like «синий» or
// «повтор», and the built-in meaning has to win. Running it first would let one
// added word quietly break colours for everyone who added it.
//
// vocab maps a word to the tag it stands for. Matching is exact and per token,
// like everything else here.
func RecogniseCustomTags(toks []Token, vocab map[string]string, d *Draft) bool {
	if len(vocab) == 0 {
		return false
	}
	norm := make(map[string]string, len(vocab))
	for word, tag := range vocab {
		w := strings.ToLower(strings.TrimSpace(word))
		if w != "" && tag != "" {
			norm[w] = strings.ToLower(strings.TrimSpace(tag))
		}
	}

	seen := map[string]bool{}
	for _, t := range d.Tags {
		seen[t] = true
	}

	found := false
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		tag, ok := norm[t.Norm]
		if !ok {
			continue
		}
		toks[i].Field = FieldTag
		found = true
		if !seen[tag] {
			seen[tag] = true
			d.Tags = append(d.Tags, tag)
		}
	}
	return found
}
