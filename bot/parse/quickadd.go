package parse

import (
	"strings"
	"time"
)

// IsTaskLine reports whether a line names itself a task — «задача», «task».
//
// Quick add uses it to skip the «what is this?» question: someone who wrote
// the word has already answered it. By token, so «задачник» is not a task.
func IsTaskLine(text string) bool {
	for _, t := range Tokenize(text) {
		if taskWords[t.Norm] {
			return true
		}
	}
	return false
}

// rangeWords sit between two days that make ONE span, not two entries:
// «с понедельника по среду», «from 14.09 to 29.09».
var rangeWords = map[string]bool{"по": true, "до": true, "to": true, "till": true, "until": true, "-": true, "–": true, "—": true}

// fillerWords are not a title on their own: a segment left holding only these
// after its day and time are read is part of its neighbour, not an entry.
var fillerWords = map[string]bool{"на": true, "в": true, "во": true, "с": true, "со": true, "и": true, "а": true, "and": true, "on": true, "at": true, "then": true}

// splitByDays cuts ONE line before its second and later day words, when every
// piece keeps a title of its own.
//
// Denis, 17.09: «tuesday morning office work wednesday midnight movie» is two
// events typed without a line break. 🔴 The result is only ever OFFERED — the
// caller asks «one entry or N?» — so a wrong cut costs one tap, and a line with
// a single day is never cut at all.
func splitByDays(line string, now time.Time) []string {
	toks := Tokenize(line)
	starts := []int{}
	for i, t := range toks {
		if !startsDay(t.Norm) {
			continue
		}
		start := i
		if j := start - 1; j >= 0 {
			if _, mod := weekdayShift[toks[j].Norm]; mod {
				start = j
			}
		}
		if j := start - 1; j >= 0 && (dayPrepositions[toks[j].Norm] || fillerWords[toks[j].Norm]) {
			start = j
		}
		if len(starts) > 0 {
			if j := start - 1; j >= 0 && rangeWords[toks[j].Norm] {
				// «… по среду» continues a span; not a new entry.
				continue
			}
			if start <= starts[len(starts)-1] {
				continue
			}
		}
		starts = append(starts, start)
	}
	if len(starts) < 2 {
		return []string{line}
	}
	if starts[0] != 0 {
		// Words before the first day belong to it: «office work tuesday …».
		starts[0] = 0
	}

	pieces := make([]string, 0, len(starts))
	for k, from := range starts {
		to := len(toks)
		if k+1 < len(starts) {
			to = starts[k+1]
		}
		words := make([]string, 0, to-from)
		for _, t := range toks[from:to] {
			words = append(words, t.Text)
		}
		piece := strings.Join(words, " ")
		if !hasOwnTitle(piece, now) {
			return []string{line}
		}
		pieces = append(pieces, piece)
	}
	return pieces
}

func startsDay(norm string) bool {
	if _, ok := weekdayWords[norm]; ok {
		return true
	}
	if _, ok := relativeDayWords[norm]; ok {
		return true
	}
	return dateTokenRe.MatchString(norm)
}

func hasOwnTitle(piece string, now time.Time) bool {
	for _, w := range strings.Fields(ParseLine(piece, now).Title) {
		if !fillerWords[strings.ToLower(w)] {
			return true
		}
	}
	return false
}
