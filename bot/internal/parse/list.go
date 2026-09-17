package parse

import (
	"regexp"
	"strings"
	"time"
)

// enumerationMarker matches the ways people number a list: «1.», «2)», «-»,
// «•», «*». The marker is removed; the number is not kept, because the order of
// the lines already carries it.
var enumerationMarker = regexp.MustCompile(`^\s*(?:\d{1,2}\s*[.)]|[-–—•*])\s+`)

// StripMarker removes one leading enumeration marker.
//
// ⚠ A digit followed by a dot is a marker only with a space after it. «16.09»
// is a date and «1.5 часа» is a quantity, and eating either would be worse than
// leaving a marker in a title.
func StripMarker(line string) string {
	return strings.TrimSpace(enumerationMarker.ReplaceAllString(line, ""))
}

// Entries splits a block into one string per entry.
//
// Newlines separate entries. Commas do too, but ONLY when there is a single
// line — «Ужин, завтра 19:00» must not become two events, and a comma inside a
// multi-line block is punctuation inside one of its lines.
func Entries(text string) []string {
	lines := []string{}
	for _, raw := range strings.Split(text, "\n") {
		if s := StripMarker(raw); s != "" {
			lines = append(lines, s)
		}
	}
	if len(lines) != 1 {
		return lines
	}

	parts := []string{}
	for _, raw := range strings.Split(lines[0], ",") {
		if s := strings.TrimSpace(raw); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 1 {
		// No line breaks, no commas — but perhaps two days, each with its own
		// title. The reference day only decides titles, not dates, so the
		// wall clock is good enough here.
		return splitByDays(parts[0], time.Now())
	}
	return parts
}

// LooksLikeList reports whether the input may be more than one entry.
//
// 🔴 It reports MAY, never IS. Denis, 15.09: «это все должно уточняться создать
// одну задачу с таким длинным описанием/названием или это список задач». So the
// answer to this function is a question put to the user, not a decision taken
// on their behalf — a long title split into five tasks is as wrong as five
// tasks glued into one title, and only they know which they meant.
func LooksLikeList(text string, now time.Time) bool {
	lines := 0
	for _, raw := range strings.Split(text, "\n") {
		if strings.TrimSpace(raw) != "" {
			lines++
		}
	}
	if lines > 1 {
		return true
	}
	if enumerationMarker.MatchString(text) {
		return true
	}

	// One line with commas. A comma next to a recognised time is punctuation
	// in a sentence — «Ужин, завтра 19:00» — not a separator.
	if strings.Contains(text, ",") {
		return !ParseLine(text, now).Draft.HasTime
	}
	return len(splitByDays(text, now)) > 1
}

// ParseEventList reads a block into one draft per entry, with day headers
// inheriting downward.
//
// Denis's example, item 5:
//
//	среда
//	10:00 завтрак
//	12:00-14:00 оркестр
//	17:00 работа
//	четверг
//	8:00 анализы
//
// 🔴 A line counts as a HEADER when it names a day, names no time, and leaves
// no title behind. All three conditions matter: «четверг» is a header, «четверг
// 8:00 анализы» is an event that also moves the day, and «четверг выходной» is
// an all-day-less event called «выходной», not a header — dropping it would
// lose something the user typed.
func ParseEventList(text string, now time.Time) []Parsed {
	entries := Entries(text)
	out := make([]Parsed, 0, len(entries))

	var day time.Time
	haveDay := false

	for _, e := range entries {
		p := ParseLine(e, now)

		// 🔴 Days in a block run forwards. «monday … tuesday … friday» written
		// on a Wednesday is next week in order, and read line by line it put
		// Friday the 18th before Monday the 21st (Mufid's block, 16.09). A bare
		// weekday that would land before the day above it moves a week on.
		// Explicit dates, «завтра» and «следующая» are left exactly as written.
		if p.Draft.HasDay && p.Draft.BareWeekday && haveDay {
			for p.Draft.Day.Before(day) {
				p.Draft.Day = p.Draft.Day.AddDate(0, 0, 7)
			}
		}

		if p.Draft.HasDay && !p.Draft.HasTime && !p.Draft.AllDay && p.Title == "" {
			day, haveDay = p.Draft.Day, true
			continue
		}

		switch {
		case p.Draft.HasDay:
			day, haveDay = p.Draft.Day, true
		case haveDay:
			p.Draft.Day, p.Draft.HasDay = day, true
		}
		out = append(out, p)
	}
	return out
}

// ParseTaskList reads a block into one task per entry.
//
// 🔴 ParseTask, not ParseLine. They read DIFFERENT vocabularies: an event line
// knows about colours and repeats, a task line knows about `!1` priorities and
// `30м` estimates. Running a task list through the event parser is what Denis
// hit on 16.09 — \«Отжаться 1ч\» became a task literally called \«Отжаться 1ч\»,
// while the same words typed as a SINGLE task parsed correctly, because that
// path always used ParseTask.
//
// No day inherits here: a task list is a list of things to do, and its lines do
// not describe a schedule the way an event block does.
func ParseTaskList(text string, now time.Time) []TaskResult {
	entries := Entries(text)
	out := make([]TaskResult, 0, len(entries))
	for _, e := range entries {
		out = append(out, ParseTask(e, now))
	}

	// 🔴 ONE line, one day: «завтра помыться, поесть, поспать» is a sentence,
	// and its day belongs to all three (Denis, 17.09: «день поставился только
	// на первую задачу»). Across SEVERAL lines nothing is inherited — that
	// stays as it was, because a list of things to do is not a schedule.
	if !strings.Contains(text, "\n") && len(out) > 1 && out[0].DueDate != nil {
		for i := range out[1:] {
			if out[i+1].DueDate == nil {
				due := *out[0].DueDate
				out[i+1].DueDate = &due
			}
		}
	}
	return out
}
