package reminders

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DigestEvent and DigestTask are the digest's view of a day: only the fields
// that reach the message. They are deliberately not the events/tasks package
// types — the digest must not grow a dependency on either, and reminders
// already imports events for occurrence expansion.
type DigestEvent struct {
	Title    string
	StartsAt time.Time
	EndsAt   time.Time
	AllDay   bool
}

type DigestTask struct {
	Title string
}

// telegramMessageLimit is Telegram's hard cap on sendMessage text. Going over
// it is not a truncation on their side, it is a 400 and no message at all —
// which is the same class of silent morning failure the empty-message bug was.
const telegramMessageLimit = 4096

// digestTitleLimit keeps one pathological title from eating the whole budget.
const digestTitleLimit = 120

// DigestText renders the daily digest.
//
// It is a pure function so the wording and the truncation are testable without
// a database: the bug it replaces (an empty message, rejected by Telegram every
// morning) survived precisely because the only way to observe it was to wait
// until 08:00 and read a FAILED row.
//
// day is the user's local midnight; loc is their zone. Times inside the events
// are UTC instants and are converted here, never printed raw.
func DigestText(day time.Time, loc *time.Location, events []DigestEvent, tasks []DigestTask) string {
	if loc == nil {
		loc = time.UTC
	}
	header := "Today — " + day.In(loc).Format("Mon, Jan 2")

	if len(events) == 0 && len(tasks) == 0 {
		return header + "\n\nNothing scheduled."
	}

	// Sort a copy: the caller's slice order is not ours to change.
	sorted := make([]DigestEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartsAt.Before(sorted[j].StartsAt) })

	var lines []string
	if len(sorted) > 0 {
		lines = append(lines, "", fmt.Sprintf("Events (%d)", len(sorted)))
		for _, e := range sorted {
			lines = append(lines, "  "+eventLine(e, loc))
		}
	}
	if len(tasks) > 0 {
		lines = append(lines, "", fmt.Sprintf("Tasks due today (%d)", len(tasks)))
		for _, t := range tasks {
			lines = append(lines, "  • "+clip(t.Title, digestTitleLimit))
		}
	}

	return assembleWithinLimit(header, lines)
}

func eventLine(e DigestEvent, loc *time.Location) string {
	title := clip(e.Title, digestTitleLimit)
	if e.AllDay {
		return "all day  " + title
	}
	start := e.StartsAt.In(loc).Format("15:04")
	end := e.EndsAt.In(loc).Format("15:04")
	if e.EndsAt.IsZero() || !e.EndsAt.After(e.StartsAt) {
		return start + "  " + title
	}
	return start + "–" + end + "  " + title
}

// assembleWithinLimit adds lines until the next one would breach Telegram's
// cap, then says how many were dropped. A digest that silently ends early is
// worse than a short one: the reader has no way to know they are missing half
// their day.
func assembleWithinLimit(header string, lines []string) string {
	var b strings.Builder
	b.WriteString(header)
	used := len([]rune(header))

	for i, line := range lines {
		remaining := len(lines) - i
		// Reserve room for the footer we would have to write if this line is
		// the one that does not fit.
		footer := fmt.Sprintf("\n\n… and %d more", remaining)
		cost := 1 + len([]rune(line)) // the newline plus the line itself

		if used+cost+len([]rune(footer)) > telegramMessageLimit {
			b.WriteString(footer)
			return b.String()
		}
		b.WriteString("\n")
		b.WriteString(line)
		used += cost
	}
	return b.String()
}

// clip shortens a title on a rune boundary. Cutting bytes would split a
// Cyrillic character in half and hand Telegram invalid UTF-8.
func clip(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit-1]) + "…"
}
