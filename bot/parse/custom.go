package parse

import (
	"strings"
	"time"
)

// Trigger is one word the user defined, and what it does.
//
// 🔴 Denis, 15.09: «Ключевые слова не обязательно теги, они должны быть как
// триггеры… при добавлении слов выбирается что оно обозначает/заменяет то есть
// какую характеристику (тег/дата/цвет/календарь и тд)».
//
// The first version of this could only make tags, which meant a word like
// «созвон» could not mean «в рабочий календарь, синим» — the one thing a
// shortcut is for.
type Trigger struct {
	Field Field
	// Value is what the field becomes. Its meaning depends on Field:
	// a tag name, a palette name, a calendar name, an RRULE, or — for a day
	// or a time — a phrase re-read by the parser itself, so «после обеда»
	// defined as 14:00 keeps working when the clock does.
	Value string
}

// TriggerFields are the characteristics a word may stand for, in the order the
// settings screen offers them.
//
// ⚠ The list is closed. A word that could mean anything would need a rule
// language, and a rule language in a Telegram keyboard is a worse product than
// six buttons.
var TriggerFields = []struct {
	Field Field
	Name  string // stored in settings; stable, not display text
	Label string
}{
	{FieldTag, "tag", "🏷 Тег"},
	{FieldColour, "colour", "🎨 Цвет"},
	{FieldCalendar, "calendar", "📁 Календарь"},
	{FieldDay, "day", "🗓 Дата"},
	{FieldTime, "time", "🕐 Время"},
	{FieldRepeat, "repeat", "🔁 Повтор"},
	{FieldAllDay, "allday", "🌅 Весь день"},
	{FieldKind, "task", "✅ Задача"},
}

// FieldByName resolves a stored field name.
func FieldByName(name string) (Field, bool) {
	for _, f := range TriggerFields {
		if f.Name == name {
			return f.Field, true
		}
	}
	return FieldNone, false
}

// FieldName is the stored name of a field, for writing back to settings.
func FieldName(f Field) string {
	for _, tf := range TriggerFields {
		if tf.Field == f {
			return tf.Name
		}
	}
	return ""
}

// FieldLabel is what the settings screen shows.
func FieldLabel(f Field) string {
	for _, tf := range TriggerFields {
		if tf.Field == f {
			return tf.Label
		}
	}
	return string(rune(f))
}

// RecogniseCustomTriggers applies the user's own words.
//
// 🔴 It runs LAST, after every built-in recogniser, and that order is the
// answer to an obvious collision: a custom word may be spelled like «синий» or
// «повтор», and the built-in meaning has to win. Running it first would let one
// added word quietly break colours for everybody who added it.
//
// ⚠ A word never overwrites a field the line already set explicitly. If the
// line says «16.09» and a custom word also means a day, the date the user typed
// stands — the shortcut is a default, not an override.
func RecogniseCustomTriggers(toks []Token, rules map[string]Trigger, now time.Time, d *Draft) bool {
	if len(rules) == 0 {
		return false
	}
	norm := make(map[string]Trigger, len(rules))
	for word, tr := range rules {
		if w := strings.ToLower(strings.TrimSpace(word)); w != "" {
			norm[w] = tr
		}
	}

	found := false
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		tr, ok := norm[t.Norm]
		if !ok {
			continue
		}
		if !applyTrigger(tr, t.Norm, now, d) {
			// The word is known but its value no longer makes sense — a colour
			// that was renamed, a broken date. Leaving the token unclaimed
			// keeps it in the title rather than deleting what the user wrote.
			continue
		}
		toks[i].Field = tr.Field
		found = true
	}
	return found
}

func applyTrigger(tr Trigger, word string, now time.Time, d *Draft) bool {
	value := strings.TrimSpace(tr.Value)

	switch tr.Field {
	case FieldTag:
		tag := strings.ToLower(value)
		if tag == "" {
			tag = word
		}
		for _, existing := range d.Tags {
			if existing == tag {
				return true
			}
		}
		d.Tags = append(d.Tags, tag)
		return true

	case FieldColour:
		if d.Colour == "" {
			d.Colour = value
		}
		return value != ""

	case FieldCalendar:
		if d.Calendar == "" {
			d.Calendar = value
		}
		return value != ""

	case FieldRepeat:
		if d.Repeat == "" && !d.RepeatAsked {
			// A word saved before v0.4.11.2 may still hold FREQ=YEARLY, which
			// the API cannot parse.
			if value == "FREQ=YEARLY" {
				value = yearlyRule
			}
			d.Repeat = value
		}
		return value != ""

	case FieldAllDay:
		setAllDay(d)
		return true

	case FieldKind:
		d.IsTask = true
		return true

	case FieldDay, FieldTime:
		// The value is a phrase, re-read now. Storing «завтра» as a phrase
		// rather than as a date is what keeps it meaning tomorrow tomorrow.
		sub := ParseLine(value, now)
		if tr.Field == FieldDay {
			if !sub.Draft.HasDay {
				return false
			}
			if !d.HasDay {
				d.Day, d.HasDay = sub.Draft.Day, true
				if sub.Draft.IsUncertain(FieldDay) {
					d.MarkUncertain(FieldDay)
				}
			}
			return true
		}
		if !sub.Draft.HasTime {
			return false
		}
		if !d.HasTime && !d.AllDay {
			setTime(d, sub.Draft.Start, sub.Draft.End, sub.Draft.HasEnd)
			if sub.Draft.IsUncertain(FieldTime) {
				d.MarkUncertain(FieldTime)
			}
		}
		return true
	}
	return false
}
