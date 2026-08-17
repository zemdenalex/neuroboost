// Package parse turns one line of chat text into an event.
//
// The bot could not create events at all until 13.08 — 🗓 Calendar answered
// "coming in a future update" — so a calendar-first product could capture a
// task from the phone but not the thing the product exists for.
//
// One line with parsing, by Denis's choice: "Ужин завтра 19:00" beats four
// button taps. The rule that makes it safe is that an unparsed line is never
// guessed at — Parse reports NeedsTime and the caller asks with buttons.
package parse

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultDuration is how long an event lasts when only a start is given.
const DefaultDuration = time.Hour

// Result is one parsed line.
type Result struct {
	Title string
	Start time.Time
	End   time.Time
	// NeedsTime is true when the line carried a title but no usable time. The
	// caller must ask rather than invent one: a silently wrong time on a
	// calendar-first product is worse than one extra tap.
	NeedsTime bool
}

var (
	// 19:00, 9:00, 19.00 — and optionally a range 19:00-20:30.
	timeRe = regexp.MustCompile(`(?:^|\s)(\d{1,2})[:.](\d{2})(?:\s*[-–—]\s*(\d{1,2})[:.](\d{2}))?(?:\s|$)`)
	// 14.08 or 14.08.2026 — day first, as everyone writes it here.
	dateRe = regexp.MustCompile(`(?:^|\s)(\d{1,2})\.(\d{1,2})(?:\.(\d{4}))?(?:\s|$)`)
)

// relativeDays maps a leading day word to an offset from today. Longest match
// wins, so "послезавтра" is not read as "завтра" with a prefix.
var relativeDays = []struct {
	word string
	days int
}{
	{"послезавтра", 2},
	{"завтра", 1},
	{"сегодня", 0},
}

// Parse reads a line like "Ужин завтра 19:00" into an event.
//
// `now` is passed in rather than read from the clock so the behaviour is
// testable and so the caller can supply the user's own timezone.
func Parse(line string, now time.Time) Result {
	text := strings.TrimSpace(line)
	if text == "" {
		return Result{NeedsTime: true}
	}

	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayGiven := false

	// Relative day word, removed from the title once consumed.
	lower := strings.ToLower(text)
	for _, rd := range relativeDays {
		if idx := strings.Index(lower, rd.word); idx >= 0 {
			day = day.AddDate(0, 0, rd.days)
			dayGiven = true
			text = text[:idx] + text[idx+len(rd.word):]
			break
		}
	}

	// Explicit date, which overrides a relative word if both appear.
	if m := dateRe.FindStringSubmatch(text); m != nil {
		d, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		year := now.Year()
		if m[3] != "" {
			year, _ = strconv.Atoi(m[3])
		}
		if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
			candidate := time.Date(year, time.Month(mo), d, 0, 0, 0, 0, now.Location())
			// A bare day.month already past this year means next year — nobody
			// schedules into the past on purpose.
			if m[3] == "" && candidate.Before(day) {
				candidate = candidate.AddDate(1, 0, 0)
			}
			day = candidate
			dayGiven = true
			text = strings.Replace(text, m[0], " ", 1)
		}
	}

	m := timeRe.FindStringSubmatch(text)
	if m == nil {
		return Result{Title: cleanTitle(text), NeedsTime: true}
	}

	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	if h > 23 || min > 59 {
		return Result{Title: cleanTitle(text), NeedsTime: true}
	}
	text = strings.Replace(text, m[0], " ", 1)

	start := time.Date(day.Year(), day.Month(), day.Day(), h, min, 0, 0, now.Location())
	// No day said and the time has already gone by: they mean tomorrow.
	if !dayGiven && start.Before(now) {
		start = start.AddDate(0, 0, 1)
	}

	end := start.Add(DefaultDuration)
	if m[3] != "" {
		eh, _ := strconv.Atoi(m[3])
		em, _ := strconv.Atoi(m[4])
		if eh <= 23 && em <= 59 {
			end = time.Date(start.Year(), start.Month(), start.Day(), eh, em, 0, 0, now.Location())
			// 23:00-01:00 crosses midnight rather than ending before it starts.
			if !end.After(start) {
				end = end.AddDate(0, 0, 1)
			}
		}
	}

	title := cleanTitle(text)
	if title == "" {
		return Result{Title: "", Start: start, End: end}
	}
	return Result{Title: title, Start: start, End: end}
}

// cleanTitle tidies what is left after the date and time were cut out of it.
func cleanTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.Trim(s, " ,.-–—")
}
