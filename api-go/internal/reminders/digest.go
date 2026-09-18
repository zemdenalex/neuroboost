package reminders

import (
	"encoding/json"
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
func DigestText(day time.Time, loc *time.Location, events []DigestEvent, tasks []DigestTask, lang string) string {
	if loc == nil {
		loc = time.UTC
	}
	ru := lang == "ru"
	header := digestHeader(day.In(loc), ru)

	if len(events) == 0 && len(tasks) == 0 {
		if ru {
			return header + "\n\nНичего не запланировано."
		}
		return header + "\n\nNothing scheduled."
	}

	// Sort a copy: the caller's slice order is not ours to change.
	sorted := make([]DigestEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartsAt.Before(sorted[j].StartsAt) })

	var lines []string
	if len(sorted) > 0 {
		title := fmt.Sprintf("Events (%d)", len(sorted))
		if ru {
			title = fmt.Sprintf("События (%d)", len(sorted))
		}
		lines = append(lines, "", title)
		for _, e := range sorted {
			lines = append(lines, "  "+eventLine(e, loc, ru))
		}
	}
	if len(tasks) > 0 {
		title := fmt.Sprintf("Tasks due today (%d)", len(tasks))
		if ru {
			title = fmt.Sprintf("Задачи на сегодня (%d)", len(tasks))
		}
		lines = append(lines, "", title)
		for _, t := range tasks {
			lines = append(lines, "  • "+clip(t.Title, digestTitleLimit))
		}
	}

	return assembleWithinLimit(header, lines, ru)
}

func eventLine(e DigestEvent, loc *time.Location, ru bool) string {
	title := clip(e.Title, digestTitleLimit)
	if e.AllDay {
		if ru {
			return "весь день  " + title
		}
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
func assembleWithinLimit(header string, lines []string, ru bool) string {
	var b strings.Builder
	b.WriteString(header)
	used := len([]rune(header))

	for i, line := range lines {
		remaining := len(lines) - i
		// Reserve room for the footer we would have to write if this line is
		// the one that does not fit.
		footer := fmt.Sprintf(moreFooter(ru), remaining)
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

// digestHeader writes the date the way each language writes a date.
//
// 🔴 Not one template with a translated month poked into it: Russian says
// «18 сентября, чт» — day, month in the genitive, then the weekday — and English
// says «Thu, Sep 18». Forcing both through one shape produces something that is
// correct in neither.
func digestHeader(d time.Time, ru bool) string {
	if !ru {
		return "Today — " + d.Format("Mon, Jan 2")
	}
	months := []string{"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	weekdays := []string{"\u0432\u0441", "\u043f\u043d", "\u0432\u0442", "\u0441\u0440", "\u0447\u0442", "\u043f\u0442", "\u0441\u0431"}
	return fmt.Sprintf("\u0421\u0435\u0433\u043e\u0434\u043d\u044f \u2014 %d %s, %s",
		d.Day(), months[int(d.Month())-1], weekdays[int(d.Weekday())])
}

// moreFooter is the "and N more" line, as a format string.
func moreFooter(ru bool) string {
	if ru {
		return "\n\n\u2026 \u0438 \u0435\u0449\u0451 %d"
	}
	return "\n\n\u2026 and %d more"
}

// DigestLang picks the language the digest is written in.
//
// 🔴 The BOT's setting wins over the account locale, because the digest is
// delivered in Telegram and the bot's language is the one the reader chose
// there. On production these already disagree for one account: locale = ru,
// settings.bot.lang = en.
func DigestLang(settings []byte, locale string) string {
	if lang := botLangFrom(settings); lang != "" {
		return lang
	}
	if locale != "" {
		return locale
	}
	return "ru"
}

// botLangFrom reads settings.bot.lang, which is where the bot keeps the
// language the user picked in Telegram.
//
// A missing or malformed blob answers "" rather than an error: the caller has a
// fallback chain, and failing a whole morning digest over an unreadable
// preference would be a much larger harm than writing it in the default
// language.
func botLangFrom(settings []byte) string {
	if len(settings) == 0 {
		return ""
	}
	var blob struct {
		Bot struct {
			Lang string `json:"lang"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(settings, &blob); err != nil {
		return ""
	}
	return blob.Bot.Lang
}
