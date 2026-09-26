package parse

import (
	"regexp"
	"time"
)

// Events over several days — v0.4.11.2.
//
// 🔴 Denis, 16.09: «нельзя создать событие на несколько дней то есть например
// “отпуск с 14.09 по 29.09”». His answer 17.09: ONE event over the span, the
// way the web already draws a multi-day event — not one per day.

var spanOpeners = map[string]bool{"с": true, "со": true, "from": true}
var spanClosers = map[string]bool{"по": true, "до": true, "to": true, "until": true, "till": true, "-": true, "–": true, "—": true}

// «14.09-29.09» typed as one token.
var dateRangeTokenRe = regexp.MustCompile(`^(\d{1,2}[./]\d{1,2}(?:[./]\d{4})?)[-–—](\d{1,2}[./]\d{1,2}(?:[./]\d{4})?)$`)

// spanWeekdays adds the genitive a span starts with («с понедельника») to the
// forms weekdayWords already knows («по среду»).
var spanWeekdays = map[string]time.Weekday{
	"понедельника": time.Monday, "вторника": time.Tuesday, "среды": time.Wednesday,
	"четверга": time.Thursday, "пятницы": time.Friday, "субботы": time.Saturday, "воскресенья": time.Sunday,
}

// spanDayAt reads one day of a span at token i: a date, a weekday, or a relative
// day. A weekday resolves to the nearest one on or after `from`.
func spanDayAt(toks []Token, i int, now, from time.Time) (time.Time, bool) {
	if i >= len(toks) || toks[i].Field != FieldNone {
		return time.Time{}, false
	}
	norm := toks[i].Norm
	if day, ok := dateFromToken(norm, now); ok {
		return day, true
	}
	wd, ok := weekdayWords[norm]
	if !ok {
		wd, ok = spanWeekdays[norm]
	}
	if ok {
		return weekdayDate(wd, 0, from), true
	}
	if off, ok := relativeDayWords[norm]; ok {
		base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return base.AddDate(0, 0, off), true
	}
	return time.Time{}, false
}

// spanClockAt reads a strict clock time at token i: «10:00». Only strict — a
// loose «10» next to a date is too easily a count.
func spanClockAt(toks []Token, i int) (time.Duration, bool) {
	if i >= len(toks) || toks[i].Field != FieldNone {
		return 0, false
	}
	m := clockRe.FindStringSubmatch(toks[i].Norm)
	if m == nil || m[2] != ":" {
		return 0, false
	}
	return clockValue(m[1], m[3])
}

// recogniseDateSpan claims «с 14.09 по 29.09», «14.09-29.09», «from monday to
// wednesday», and «с 14.10 10:00 по 16.10 18:00».
func recogniseDateSpan(toks []Token, now time.Time, d *Draft) bool {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i := range toks {
		if toks[i].Field != FieldNone {
			continue
		}

		// One token: «14.09-29.09».
		if m := dateRangeTokenRe.FindStringSubmatch(toks[i].Norm); m != nil {
			from, okFrom := dateFromToken(m[1], now)
			to, okTo := dateFromToken(m[2], now)
			if okFrom && okTo {
				applySpan(d, from, to, 0, 0, false, false)
				toks[i].Field = FieldDay
				return true
			}
		}

		j := i
		if spanOpeners[toks[i].Norm] {
			j = i + 1
		}
		from, ok := spanDayAt(toks, j, now, today)
		if !ok {
			continue
		}
		k := j + 1
		startClock, hasStart := spanClockAt(toks, k)
		if hasStart {
			k++
		}
		if k >= len(toks) || toks[k].Field != FieldNone || !spanClosers[toks[k].Norm] {
			continue
		}
		// ⚠ «сдать отчёт до 01.12» never reaches here: a span needs a day on
		// BOTH sides of the closer. A guard for «до without с» was tried and
		// removed — sabotaging it changed no test, because it guarded nothing.
		to, ok := spanDayAt(toks, k+1, now, from)
		if !ok {
			continue
		}
		last := k + 1
		endClock, hasEnd := spanClockAt(toks, last+1)
		if hasEnd {
			last++
		}

		applySpan(d, from, to, startClock, endClock, hasStart, hasEnd)
		for x := i; x <= last; x++ {
			toks[x].Field = FieldDay
		}
		return true
	}
	return false
}

func applySpan(d *Draft, from, to time.Time, startClock, endClock time.Duration, hasStart, hasEnd bool) {
	d.Day, d.HasDay = from, true
	d.EndDay = to
	if to.Before(from) {
		// 🔴 Marked, not swapped: which of the two dates is the typo is not
		// something the bot can know.
		d.MarkUncertain(FieldDay)
	}
	if !hasStart {
		setAllDay(d)
		return
	}
	days := time.Duration(0)
	if to.After(from) {
		days = to.Sub(from).Round(24 * time.Hour)
	}
	d.Start, d.HasTime = startClock, true
	if !hasEnd {
		endClock = startClock
	}
	d.End, d.HasEnd = days+endClock, true
}
