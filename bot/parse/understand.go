package parse

import (
	"strings"
	"time"
)

// This file is the part of «what did the user mean by this line» that sits
// ABOVE the recogniser pipeline: the user's calendars, their own words, their
// reminder presets, and the rule that decides whether a line needs a question.
//
// 🔴 It lives here, in the parser, and not in the bot's handlers, because two
// programs ask the question: the bot, and the API's POST /api/parse that the
// web's quick-add row calls (Denis, 26.09: «one parser, API endpoint»). The
// order below was the bot's parseIntoDraft; a second copy of it in api-go would
// be a second dialect, and a typed line would start meaning different things
// in the chat and on the page.

// Keyword is one word the user defined: which characteristic it sets, and to
// what. Field is the stored NAME («tag», «colour», «calendar»…), not a Field:
// a string in the settings blob survives a renumbering of the enum.
type Keyword struct {
	Field string
	Value string
}

// Calendar is one calendar the user can write to, as the line may name it.
type Calendar struct {
	ID   string
	Name string
}

// Vocabulary is everything about the user that the pipeline cannot know.
// A zero Vocabulary is valid: no calendars, no words, no presets.
type Vocabulary struct {
	Calendars []Calendar
	Keywords  map[string]Keyword
	Presets   map[string][]int
}

// Understood is a line read with the user's vocabulary.
type Understood struct {
	Draft        Draft
	Tokens       []Token
	CalendarID   string
	CalendarName string
	Title        string
}

// Understand runs the pipeline and then everything that needs the user's data.
//
// 🔴 The order is load-bearing:
//  1. the calendar named in the line;
//  2. the user's own words LAST, after every built-in recogniser and after the
//     calendar — a custom word spelled like «синий» or «повтор» must not take
//     the built-in meaning away from the person who added it;
//  3. a calendar a custom word named, resolved by name;
//  4. reminder presets;
//  5. the title, recomputed AFTER all of it — recomputing earlier would leave
//     the calendar's name in the title.
func Understand(line string, now time.Time, v Vocabulary) Understood {
	p := ParseLine(line, now)
	u := Understood{Draft: p.Draft, Tokens: p.Tokens}

	if len(v.Calendars) > 0 {
		names := make([]string, len(v.Calendars))
		for i, c := range v.Calendars {
			names[i] = c.Name
		}
		if idx, ok := RecogniseCalendar(p.Tokens, names, &u.Draft); ok {
			u.CalendarID, u.CalendarName = v.Calendars[idx].ID, v.Calendars[idx].Name
		}
	}

	if len(v.Keywords) > 0 {
		rules := make(map[string]Trigger, len(v.Keywords))
		for word, kw := range v.Keywords {
			field, ok := FieldByName(kw.Field)
			if !ok {
				// A characteristic this build does not know — written by a
				// newer version, or renamed. Skipping it leaves the word in
				// the title, which is visible; guessing a field would not be.
				continue
			}
			rules[word] = Trigger{Field: field, Value: kw.Value}
		}
		RecogniseCustomTriggers(p.Tokens, rules, now, &u.Draft)
	}

	if u.CalendarID == "" && u.Draft.Calendar != "" {
		for _, c := range v.Calendars {
			if strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(u.Draft.Calendar)) {
				u.CalendarID, u.CalendarName = c.ID, c.Name
				break
			}
		}
	}

	if len(v.Presets) > 0 {
		RecogniseReminderPreset(p.Tokens, v.Presets, &u.Draft)
	}

	u.Title = Title(p.Tokens)
	return u
}

// Missing names what the card still has to ask before an event can be
// created, in the order the user should be asked: "title", "freq", "date",
// "span", "time" — or "" when nothing is missing.
//
// 🔴 A bare «повтор» is a question, not a default. Denis, 15.09: «поскольу
// повтор я не написал частоту, тоже должен уточнить». Creating a one-off
// because no frequency was given would be a silent answer to a question he
// explicitly asked to be asked.
func Missing(title string, d Draft) string {
	switch {
	case strings.TrimSpace(title) == "":
		return "title"
	case d.RepeatAsked && d.Repeat == "":
		return "freq"
	case !d.HasDay:
		return "date"
	case !d.EndDay.IsZero() && d.EndDay.Before(d.Day):
		// 🔴 Denis, 17.09: ⚠ was shown and ✅ created it anyway; then the plain
		// day question offered three buttons and no way to fix a SPAN.
		return "span"
	case !d.HasTime && !d.AllDay:
		return "time"
	default:
		return ""
	}
}

// PlainTask is a line that needs no question and is saved as a task at once
// (Denis, 23.09: «seamless task creation»): no clock time (task or event is a
// real choice there — a timed task is bound to an event), not a list (one or
// many is a real choice), and no «повтор» without a frequency (the card
// promises to ask). Returns what the task parser made of it.
//
// ⚠ The task path does not read the user's own words: the bot's quick save
// never did, and «exactly as the bot» includes that.
func PlainTask(line string, now time.Time) (TaskResult, bool) {
	if ParseLine(line, now).Draft.HasTime || LooksLikeList(line, now) {
		return TaskResult{}, false
	}
	r := ParseTask(line, now)
	if r.RepeatAsked || strings.TrimSpace(r.Title) == "" {
		return TaskResult{}, false
	}
	return r, true
}

// Bounds turns a draft's day and offsets into two instants. An all-day event
// spans local midnight to local midnight; anything else runs an hour unless an
// end was given.
func Bounds(d Draft) (time.Time, time.Time) {
	if d.AllDay {
		// An all-day event ends at the midnight AFTER its last day — one day
		// or a span, the same convention.
		last := d.Day
		if !d.EndDay.IsZero() && d.EndDay.After(last) {
			last = d.EndDay
		}
		return d.Day, last.AddDate(0, 0, 1)
	}
	start := d.StartsAt()
	if d.HasEnd {
		return start, d.EndsAt()
	}
	return start, start.Add(time.Hour)
}

// KeywordsFromSettings reads the user's own words from the settings blob
// (`bot.keywords`, word → {field, value}).
//
// A missing or malformed section is an empty vocabulary, not an error.
//
// ⚠ A bare string is the FIRST shape this feature shipped with, where every
// word was a tag. It is still read as one. Dropping it would silently empty the
// vocabulary of anyone who used the version that stored it.
func KeywordsFromSettings(settings map[string]any) map[string]Keyword {
	out := map[string]Keyword{}
	bot, ok := settings["bot"].(map[string]any)
	if !ok {
		return out
	}
	words, ok := bot["keywords"].(map[string]any)
	if !ok {
		return out
	}

	for word, raw := range words {
		w := strings.ToLower(strings.TrimSpace(word))
		if w == "" {
			continue
		}
		switch v := raw.(type) {
		case string:
			if s := strings.ToLower(strings.TrimSpace(v)); s != "" {
				out[w] = Keyword{Field: "tag", Value: s}
			}
		case map[string]any:
			field, _ := v["field"].(string)
			value, _ := v["value"].(string)
			field = strings.ToLower(strings.TrimSpace(field))
			if field == "" {
				continue
			}
			out[w] = Keyword{Field: field, Value: strings.TrimSpace(value)}
		}
	}
	return out
}

// PresetsFromSettings reads the user's named reminder presets
// (`reminders.presets`, name → offsets in minutes).
//
// ⚠ A preset naming an EMPTY list is a real answer — «без» means stay silent —
// so an empty slice is kept, and only a value of the wrong shape is skipped.
func PresetsFromSettings(settings map[string]any) map[string][]int {
	out := map[string][]int{}
	reminders, ok := settings["reminders"].(map[string]any)
	if !ok {
		return out
	}
	presets, ok := reminders["presets"].(map[string]any)
	if !ok {
		return out
	}
	for name, raw := range presets {
		list, ok := raw.([]any)
		if !ok {
			continue
		}
		offsets := make([]int, 0, len(list))
		bad := false
		for _, v := range list {
			n, ok := v.(float64) // every number out of encoding/json is a float64
			if !ok {
				bad = true
				break
			}
			offsets = append(offsets, int(n))
		}
		if bad {
			continue
		}
		if n := strings.ToLower(strings.TrimSpace(name)); n != "" {
			out[n] = offsets
		}
	}
	return out
}
