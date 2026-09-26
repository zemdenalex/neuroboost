package parse

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TaskResult is one parsed task line. Every optional field is a pointer so that
// "not stated" and "stated as zero" stay different answers.
//
// 🔴 Priority is the reason this matters. 0 is Buffer, a real priority a user
// can choose, and 5 is "if possible". A plain int would make an unstated
// priority indistinguishable from Buffer, and the API would store the wrong one
// without anything looking broken.
type TaskResult struct {
	Title            string
	Priority         *int
	DueDate          *time.Time
	EstimatedMinutes *int
	Tags             []string
	// Rrule is the repeat, in the API's grammar, or "" for a one-off task.
	Rrule string
	// RepeatAsked is «повтор» with no frequency: a question the card must ask,
	// the same way an event's card does. Never a guess, never a word left in
	// the title.
	RepeatAsked bool
}

var (
	// !0 … !5, standing alone. The trailing boundary is what stops "!срочно"
	// from being read as a priority and having its word eaten.
	priorityRe = regexp.MustCompile(`(?:^|\s)!([0-5])(?:\s|$)`)
	// 30м, 90м, 1ч, 2ч. Minutes and hours, nothing finer — this is a phone.
	estimateRe = regexp.MustCompile(`(?:^|\s)(\d{1,3})\s*(м|мин|ч|час)(?:\s|$)`)
	// #тег. \p{L} rather than \w: every tag here is Cyrillic, and \w is ASCII.
	tagRe = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)
)

// dueDatePrepositions are swallowed with a due day: «на завтра», «for tomorrow».
var dueDatePrepositions = map[string]bool{"на": true, "к": true, "до": true, "for": true, "by": true}

// ParseTask reads a line like "позвонить в банк завтра 30м !1 #дела".
//
// `now` is a parameter rather than a clock read so the behaviour is testable
// and so the caller supplies the user's own timezone — same contract as Parse.
//
// 🔴 Whatever is not recognised stays in the title. Silently dropping a word
// the user typed is worse than not parsing it: they lose text and never learn
// why. Only a marker that matched in full is cut out.
func ParseTask(line string, now time.Time) TaskResult {
	res := TaskResult{Tags: []string{}}
	text := strings.TrimSpace(line)
	if text == "" {
		return res
	}

	if m := tagRe.FindAllStringSubmatch(text, -1); m != nil {
		seen := map[string]bool{}
		for _, g := range m {
			tag := strings.ToLower(g[1])
			if !seen[tag] {
				seen[tag] = true
				res.Tags = append(res.Tags, tag)
			}
		}
		text = tagRe.ReplaceAllString(text, " ")
	}

	if m := priorityRe.FindStringSubmatch(text); m != nil {
		p, _ := strconv.Atoi(m[1])
		res.Priority = &p
		text = strings.Replace(text, m[0], " ", 1)
	}

	if m := estimateRe.FindStringSubmatch(text); m != nil {
		n, _ := strconv.Atoi(m[1])
		if strings.HasPrefix(m[2], "ч") {
			n *= 60
		}
		if n > 0 {
			res.EstimatedMinutes = &n
			text = strings.Replace(text, m[0], " ", 1)
		}
	}

	// The day words and explicit dates are already solved for events; reuse
	// that shape rather than growing a second dialect of the same thing.
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 🔴 By token, not by substring. «Index(lower, "завтра")» found «завтра»
	// inside «завтрак» — the exact defect that once titled an event «к» — and
	// it survived here after the event parser was rebuilt on tokens.
	toks := Tokenize(text)
	for i, t := range toks {
		if taskWords[t.Norm] {
			// «задача» named the kind; quick add already acted on it, and it
			// is not part of what needs doing.
			toks[i].Field = FieldKind
			continue
		}
		if res.DueDate != nil {
			continue
		}
		if offset, ok := relativeDayWords[t.Norm]; ok && offset >= 0 {
			d := day.AddDate(0, 0, offset)
			res.DueDate = &d
			toks[i].Field = FieldDay
			// «на завтра», «for tomorrow» — the preposition goes with the day.
			if i > 0 && toks[i-1].Field == FieldNone && dueDatePrepositions[toks[i-1].Norm] {
				toks[i-1].Field = FieldDay
			}
		}
	}
	text = Title(toks)
	if res.DueDate == nil {
		if m := dateRe.FindStringSubmatch(text); m != nil {
			d, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			year := now.Year()
			if m[3] != "" {
				year, _ = strconv.Atoi(m[3])
			}
			if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
				candidate := time.Date(year, time.Month(mo), d, 0, 0, 0, 0, now.Location())
				if m[3] == "" && candidate.Before(day) {
					candidate = candidate.AddDate(1, 0, 0)
				}
				res.DueDate = &candidate
				text = strings.Replace(text, m[0], " ", 1)
			}
		}
	}

	res.Title = cleanTitle(text)
	res.Title, res.Rrule, res.RepeatAsked = taskRepeat(res.Title)
	return res
}

// taskRepeat reads the repetition words out of a task's title.
//
// 🔴 It calls recogniseRepeat — the SAME recogniser the event parser runs — and
// not a second vocabulary written beside it. Until 20.09 tasks had none at all,
// so «повтор пить таблетки» meant one thing after 📅 and another after ➕. Two
// lists would have closed that gap for a week and reopened it the first time a
// word was added to one of them.
//
// Runs last, on the title that survived every other marker: by then «!1», «5м»
// and «#тег» are already gone, so the recogniser only ever sees words. And it
// honours ParseTask's contract — only the tokens the recogniser CLAIMED are cut;
// anything it did not understand stays in the title where the user can see it.
func taskRepeat(title string) (string, string, bool) {
	toks := Tokenize(title)
	var d Draft
	if !recogniseRepeat(toks, &d) {
		return title, "", false
	}
	var keep []string
	for _, t := range toks {
		if t.Field != FieldRepeat {
			keep = append(keep, t.Text)
		}
	}
	return cleanTitle(strings.Join(keep, " ")), d.RRule(), d.RepeatAsked
}
