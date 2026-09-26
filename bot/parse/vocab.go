package parse

import (
	"strings"
	"time"
)

// ---------------------------------------------------------------- repeat ---

// repeatOnly are the words that say "this repeats" without saying how often.
//
// 🔴 They set RepeatAsked rather than a default frequency. Denis, 15.09:
// «поскольу повтор я не написал частоту, тоже должен уточнить». Picking
// FREQ=WEEKLY here would be a guess written silently into his calendar, and
// the repeat is the one field where a wrong guess multiplies.
var repeatOnly = map[string]bool{
	"повтор": true, "повторять": true, "повторяется": true,
	"повторяющееся": true, "повторное": true,
	"repeat": true, "recurring": true, "repeats": true,
}

var namedFrequency = map[string]string{
	"ежедневно":   "FREQ=DAILY",
	"daily":       "FREQ=DAILY",
	"еженедельно": "FREQ=WEEKLY",
	"weekly":      "FREQ=WEEKLY",
	"ежемесячно":  "FREQ=MONTHLY",
	"monthly":     "FREQ=MONTHLY",
	"ежегодно":    yearlyRule,
	"yearly":      yearlyRule,
	"annually":    yearlyRule,
}

// everyWords open a two-word frequency: «каждый день», «каждую неделю»,
// «каждый вторник».
var everyWords = map[string]bool{
	"каждый": true, "каждую": true, "каждое": true, "каждые": true, "кажд": true,
	"every": true,
}

var everyPeriod = map[string]string{
	"день": "FREQ=DAILY", "дня": "FREQ=DAILY", "day": "FREQ=DAILY",
	"неделю": "FREQ=WEEKLY", "недели": "FREQ=WEEKLY", "week": "FREQ=WEEKLY",
	"месяц": "FREQ=MONTHLY", "месяца": "FREQ=MONTHLY", "month": "FREQ=MONTHLY",
	"год": yearlyRule, "года": yearlyRule, "year": yearlyRule,
}

var bydayCode = map[time.Weekday]string{
	time.Monday: "MO", time.Tuesday: "TU", time.Wednesday: "WE",
	time.Thursday: "TH", time.Friday: "FR", time.Saturday: "SA", time.Sunday: "SU",
}

// recogniseRepeat claims repetition words.
//
// 🔴 It runs BEFORE the weekday recogniser, and «каждый вторник» is why. That
// phrase carries two facts — the rule repeats on Tuesdays, and the first one is
// the coming Tuesday — so this function claims only «каждый», reads the weekday
// for the BYDAY code, and leaves the weekday token itself for recogniseWeekday
// to turn into a date. Claiming both would lose the start day; running after
// the weekday recogniser would leave «каждый» stranded in the title.
func recogniseRepeat(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}

		// «раз в 3 дня», «каждые 2 недели», «every 2 weeks», «через день».
		if rule, width, ok := periodAt(toks, i); ok {
			d.Repeat = rule
			for k := i; k < i+width; k++ {
				toks[k].Field = FieldRepeat
			}
			return true
		}

		if freq, ok := namedFrequency[t.Norm]; ok {
			d.Repeat = freq
			toks[i].Field = FieldRepeat
			return true
		}

		if everyWords[t.Norm] && i+1 < len(toks) {
			next := toks[i+1]
			if freq, ok := everyPeriod[next.Norm]; ok {
				d.Repeat = freq
				toks[i].Field, toks[i+1].Field = FieldRepeat, FieldRepeat
				return true
			}
			if _, ok := weekdayWords[next.Norm]; ok {
				// 🔴 Not BYDAY. The API rejects every RRULE key but FREQ,
				// INTERVAL, COUNT and UNTIL, and an event it cannot parse is
				// shown once and never repeats. Weekly from a Tuesday IS
				// every Tuesday — the weekday recogniser picks that start.
				d.Repeat = "FREQ=WEEKLY"
				toks[i].Field = FieldRepeat // the weekday stays for the day recogniser
				return true
			}
		}

		if repeatOnly[t.Norm] {
			d.RepeatAsked = true
			toks[i].Field = FieldRepeat
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- all day ---

var allDayOpeners = map[string]bool{"весь": true, "целый": true, "all": true, "на": true}
var allDayNouns = map[string]bool{"день": true, "day": true}

// recogniseAllDay claims «весь день» and its variants.
//
// ⚠ Two words, never one. «день» alone is an ordinary noun, and claiming it
// would retitle «хороший день» to «хороший» — the same class of damage as the
// substring bug this rewrite exists to end.
func recogniseAllDay(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}

		if t.Norm == "allday" || t.Norm == "all-day" {
			toks[i].Field = FieldAllDay
			setAllDay(d)
			return true
		}

		if !allDayOpeners[t.Norm] || i+1 >= len(toks) {
			continue
		}
		j := i + 1
		if t.Norm == "на" {
			// «на» is a preposition, not a marker: it counts only when «весь»
			// or «целый» follows it. Without this, «перенести на день» would
			// become an all-day event.
			if toks[j].Field != FieldNone || (toks[j].Norm != "весь" && toks[j].Norm != "целый") || j+1 >= len(toks) {
				continue
			}
			j++
		}
		if toks[j].Field != FieldNone || !allDayNouns[toks[j].Norm] {
			continue
		}
		for k := i; k <= j; k++ {
			toks[k].Field = FieldAllDay
		}
		setAllDay(d)
		return true
	}
	return false
}

// setAllDay drops any time that was already found. An all-day event with a
// start time is two answers to one question, and the calendar would honour the
// one the user did not mean.
func setAllDay(d *Draft) {
	d.AllDay = true
	d.HasTime = false
	d.HasEnd = false
	d.Start, d.End = 0, 0
}

// ------------------------------------------------------------------- kind ---

var taskWords = map[string]bool{"задача": true, "задачу": true, "задачи": true, "task": true, "todo": true}

func recogniseKind(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		if taskWords[t.Norm] {
			d.IsTask = true
			toks[i].Field = FieldKind
			return true
		}
	}
	return false
}

// ----------------------------------------------------------------- colour ---

// colourStems maps a palette name to the Russian adjective stems that mean it.
// The palette names come from web/src/lib/calendar/palette.ts and nothing else
// is accepted there — a colour stored as «синий» is saved, shown as set, and
// never painted.
var colourStems = map[string][]string{
	"blue":   {"син"},
	"cyan":   {"голуб"},
	"green":  {"зелён", "зелен"},
	"amber":  {"жёлт", "желт", "оранжев"},
	"red":    {"красн"},
	"pink":   {"розов"},
	"violet": {"фиолетов", "сиренев"},
	"slate":  {"сер"},
}

// adjectiveEndings are the forms an adjective actually arrives in.
//
// 🔴 A stem PREFIX check would be shorter and wrong: «сер» opens «серьёзно»,
// «сердце» and «середина». Expanding stem+ending into exact words keeps the
// match exact, which is the whole discipline of this package.
var adjectiveEndings = []string{
	"ый", "ий", "ой", "ое", "ее", "ая", "яя", "ым", "им", "ом", "ем",
	"ого", "его", "ую", "юю", "ые", "ие", "ых", "их",
}

var colourWords = map[string]string{
	"blue": "blue", "cyan": "cyan", "teal": "cyan",
	"green": "green", "amber": "amber", "yellow": "amber", "orange": "amber",
	"red": "red", "pink": "pink", "purple": "violet", "violet": "violet",
	"grey": "slate", "gray": "slate", "slate": "slate",
}

func init() {
	for palette, stems := range colourStems {
		for _, stem := range stems {
			for _, end := range adjectiveEndings {
				colourWords[stem+end] = palette
			}
		}
	}
}

func recogniseColour(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		if name, ok := colourWords[t.Norm]; ok {
			d.Colour = name
			toks[i].Field = FieldColour
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------- tags ---

// recogniseTags claims every #tag, lowercased and de-duplicated in the order
// they were written.
func recogniseTags(toks []Token, d *Draft) bool {
	found := false
	seen := map[string]bool{}
	for _, tag := range d.Tags {
		seen[tag] = true
	}
	for i, t := range toks {
		if t.Field != FieldNone || !strings.HasPrefix(t.Norm, "#") {
			continue
		}
		tag := strings.TrimPrefix(t.Norm, "#")
		if tag == "" {
			continue
		}
		toks[i].Field = FieldTag
		found = true
		if !seen[tag] {
			seen[tag] = true
			d.Tags = append(d.Tags, tag)
		}
	}
	return found
}
