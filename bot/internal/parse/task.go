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
	lower := strings.ToLower(text)
	for _, rd := range relativeDays {
		if idx := strings.Index(lower, rd.word); idx >= 0 {
			d := day.AddDate(0, 0, rd.days)
			res.DueDate = &d
			text = text[:idx] + text[idx+len(rd.word):]
			break
		}
	}
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
	return res
}
