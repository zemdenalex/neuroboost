package reminders

import (
	"fmt"
	"strings"
	"time"
)

// Notification text, as the user actually reads it.
//
// Until 13.08 the stored message was the bare title — insertReminder was handed
// `event.Title` and nothing else. A push saying only "YIIIIPIIIIEEEEEE" does not
// say what it is, when the thing happens, or that it is a reminder at all.
//
// Two lines, by Denis's choice:
//
//	⏰ Через 15 минут — 01:15
//	YIIIIPIIIIEEEEEE
//
// The first line is context, the second is the thing itself. Snoozing rewrites
// only the first line, which is why they are kept separate — see SnoozedText.
//
// Plain text on purpose: the notifier sends with no ParseMode, so a title
// containing *, _ or [ cannot become markup. Do not add ParseMode without
// escaping titles — that combination was a live phishing vector in this repo.

const (
	titleLineSep = "\n"
	taskMarker   = "📋 "
)

// ReminderText renders an event or task reminder.
//
// `kind` is the reminder's source kind ("EVENT" / "TASK"). `occurrence` is when
// the thing actually happens, in UTC; `minutesBefore` is the configured offset,
// so the phrasing matches what the user asked for rather than being recomputed
// from a clock that may have drifted.
func ReminderText(kind, title string, occurrence time.Time, minutesBefore int, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	local := occurrence.In(loc)
	fireLocal := local.Add(-time.Duration(minutesBefore) * time.Minute)

	lead := leadPhrase(fireLocal, local, minutesBefore)
	if kind == "TASK" {
		return "⏰ Срок " + lowerFirst(lead) + titleLineSep + taskMarker + title
	}
	return "⏰ " + lead + titleLineSep + title
}

// leadPhrase describes when the thing happens, relative to when the reminder
// fires. Same day says how long is left; another day names the day, because
// "через 1440 минут" is not something anyone can act on.
func leadPhrase(fire, occurrence time.Time, minutesBefore int) string {
	hhmm := occurrence.Format("15:04")

	fireDay := time.Date(fire.Year(), fire.Month(), fire.Day(), 0, 0, 0, 0, fire.Location())
	occDay := time.Date(occurrence.Year(), occurrence.Month(), occurrence.Day(), 0, 0, 0, 0, occurrence.Location())
	daysApart := int(occDay.Sub(fireDay).Hours() / 24)

	switch {
	case daysApart == 1:
		return fmt.Sprintf("Завтра в %s", hhmm)
	case daysApart > 1:
		return fmt.Sprintf("%d %s в %s", occurrence.Day(), monthGenitive(occurrence.Month()), hhmm)
	}

	// Same day (or the offset is zero/negative, which reads as "now").
	switch {
	case minutesBefore <= 0:
		return fmt.Sprintf("Сейчас — %s", hhmm)
	case minutesBefore < 60:
		return fmt.Sprintf("Через %d %s — %s", minutesBefore,
			pluralRu(minutesBefore, "минуту", "минуты", "минут"), hhmm)
	default:
		hours := minutesBefore / 60
		mins := minutesBefore % 60
		if mins == 0 {
			return fmt.Sprintf("Через %d %s — %s", hours,
				pluralRu(hours, "час", "часа", "часов"), hhmm)
		}
		return fmt.Sprintf("Через %d %s %d %s — %s",
			hours, pluralRu(hours, "час", "часа", "часов"),
			mins, pluralRu(mins, "минуту", "минуты", "минут"), hhmm)
	}
}

// SnoozedText rewrites the context line of an existing notification.
//
// Snoozing copies the original row's message, so without this a reminder
// postponed by ten minutes would still insist "Через 15 минут" — the one claim
// snoozing just made false. The title line is preserved verbatim, including a
// title that happens to contain a newline.
func SnoozedText(original string, minutes int) string {
	body := original
	if i := strings.Index(original, titleLineSep); i >= 0 {
		body = original[i+len(titleLineSep):]
	}
	return fmt.Sprintf("😴 Отложено на %d %s", minutes,
		pluralRu(minutes, "минуту", "минуты", "минут")) + titleLineSep + body
}

// pluralRu picks the Russian numeric form: 1 минута, 2 минуты, 5 минут.
func pluralRu(n int, one, few, many string) string {
	if n < 0 {
		n = -n
	}
	if n%100 >= 11 && n%100 <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}

func monthGenitive(m time.Month) string {
	names := [...]string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	return names[int(m)-1]
}

func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}
