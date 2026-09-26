package parse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Repeat periods and ends — v0.4.11.2.
//
// 🔴 Denis, 16.09: «кому-то нужно пить таблетки раз в 3 дня, или кормить змея
// раз в 3 недели… и даже нельзя указать сколько раз повторять».
//
// 🔴 The API is the limit on what may be written. api-go/internal/events/
// recurrence.go accepts FREQ=DAILY|WEEKLY|MONTHLY with INTERVAL, COUNT and
// UNTIL (a plain date), and rejects anything else — and a rejected rule is not
// an error anyone sees: the event is shown once and never repeats. So a year is
// twelve months, and a weekday is weekly from that weekday.

// yearlyRule is «every year» in the API's vocabulary.
const yearlyRule = "FREQ=MONTHLY;INTERVAL=12"

type periodUnit struct {
	freq string
	mult int
}

var periodUnits = map[string]periodUnit{
	"день": {"DAILY", 1}, "дня": {"DAILY", 1}, "дней": {"DAILY", 1}, "day": {"DAILY", 1}, "days": {"DAILY", 1},
	"неделю": {"WEEKLY", 1}, "неделя": {"WEEKLY", 1}, "недели": {"WEEKLY", 1}, "недель": {"WEEKLY", 1}, "week": {"WEEKLY", 1}, "weeks": {"WEEKLY", 1},
	"месяц": {"MONTHLY", 1}, "месяца": {"MONTHLY", 1}, "месяцев": {"MONTHLY", 1}, "month": {"MONTHLY", 1}, "months": {"MONTHLY", 1},
	"год": {"MONTHLY", 12}, "года": {"MONTHLY", 12}, "лет": {"MONTHLY", 12}, "year": {"MONTHLY", 12}, "years": {"MONTHLY", 12},
}

// FreqRule turns a frequency button (DAILY, WEEKLY, MONTHLY, YEARLY) into a
// rule the API can parse.
func FreqRule(freq string) string {
	if strings.EqualFold(freq, "YEARLY") {
		return yearlyRule
	}
	return "FREQ=" + strings.ToUpper(freq)
}

func periodRule(u periodUnit, n int) string {
	interval := n * u.mult
	if interval <= 1 {
		return "FREQ=" + u.freq
	}
	return fmt.Sprintf("FREQ=%s;INTERVAL=%d", u.freq, interval)
}

// periodAt reads a period starting at token i and returns the rule and how many
// tokens it spans.
func periodAt(toks []Token, i int) (string, int, bool) {
	free := func(k int) bool { return k < len(toks) && toks[k].Field == FieldNone }
	number := func(k int) (int, bool) {
		if !free(k) {
			return 0, false
		}
		n, err := strconv.Atoi(toks[k].Norm)
		return n, err == nil && n > 0 && n < 1000
	}
	unit := func(k int) (periodUnit, bool) {
		if !free(k) {
			return periodUnit{}, false
		}
		u, ok := periodUnits[toks[k].Norm]
		return u, ok
	}

	switch t := toks[i].Norm; {
	case t == "раз" && free(i+1) && (toks[i+1].Norm == "в" || toks[i+1].Norm == "во"):
		// «раз в 3 дня», «раз в неделю»
		if n, ok := number(i + 2); ok {
			if u, ok := unit(i + 3); ok {
				return periodRule(u, n), 4, true
			}
		}
		if u, ok := unit(i + 2); ok {
			return periodRule(u, 1), 3, true
		}
	case everyWords[t]:
		// «каждые 2 дня», «every 2 weeks». The one-word form («каждый день»)
		// is left to everyPeriod, which already knows it.
		if n, ok := number(i + 1); ok {
			if u, ok := unit(i + 2); ok {
				return periodRule(u, n), 3, true
			}
		}
	case t == "через" && free(i+1) && toks[i+1].Norm == "день":
		// «через день» — every other day. «через неделю» is NOT read: it
		// means «in a week» far more often than «every other week».
		return "FREQ=DAILY;INTERVAL=2", 2, true
	}
	return "", 0, false
}

// lineRepeats reports whether anything in the line says it repeats. The end of
// a series is read only then: «отжаться 12 раз» is twelve push-ups.
func lineRepeats(toks []Token) bool {
	for i, t := range toks {
		if _, _, ok := periodAt(toks, i); ok {
			return true
		}
		if _, ok := namedFrequency[t.Norm]; ok {
			return true
		}
		if repeatOnly[t.Norm] {
			return true
		}
		if everyWords[t.Norm] && i+1 < len(toks) {
			if _, ok := everyPeriod[toks[i+1].Norm]; ok {
				return true
			}
			if _, ok := weekdayWords[toks[i+1].Norm]; ok {
				return true
			}
		}
	}
	return false
}

var countWords = map[string]bool{"раз": true, "раза": true, "times": true}
var untilWords = map[string]bool{"до": true, "until": true, "till": true}

// recogniseRepeatEnd claims «10 раз» and «до 01.12» in a line that repeats.
func recogniseRepeatEnd(toks []Token, now time.Time, d *Draft) bool {
	if !lineRepeats(toks) {
		return false
	}
	found := false
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Field != FieldNone || toks[i+1].Field != FieldNone {
			continue
		}
		// «10 раз», but not the «раз» of «раз в неделю».
		if n, err := strconv.Atoi(toks[i].Norm); err == nil && n > 0 && countWords[toks[i+1].Norm] {
			if i+2 < len(toks) && (toks[i+2].Norm == "в" || toks[i+2].Norm == "во") {
				continue
			}
			d.RepeatCount, d.RepeatUntil = n, time.Time{}
			toks[i].Field, toks[i+1].Field = FieldRepeat, FieldRepeat
			found = true
			continue
		}
		if untilWords[toks[i].Norm] {
			if day, ok := dateFromToken(toks[i+1].Norm, now); ok {
				d.RepeatUntil, d.RepeatCount = day, 0
				toks[i].Field, toks[i+1].Field = FieldRepeat, FieldRepeat
				found = true
			}
		}
	}
	return found
}

// dateFromToken reads «01.12» or «01.12.2026» the way the date recogniser does:
// a day already gone this year means next year.
func dateFromToken(norm string, now time.Time) (time.Time, bool) {
	m := dateTokenRe.FindStringSubmatch(norm)
	if m == nil {
		return time.Time{}, false
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[3])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, false
	}
	year := now.Year()
	if m[4] != "" {
		year, _ = strconv.Atoi(m[4])
	}
	candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, now.Location())
	if m[4] == "" && candidate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
		candidate = candidate.AddDate(1, 0, 0)
	}
	return candidate, true
}

// RRule is the rule as the API stores it: the frequency, then the end.
func (d Draft) RRule() string {
	if d.Repeat == "" {
		return ""
	}
	switch {
	case d.RepeatCount > 0:
		return fmt.Sprintf("%s;COUNT=%d", d.Repeat, d.RepeatCount)
	case !d.RepeatUntil.IsZero():
		return d.Repeat + ";UNTIL=" + d.RepeatUntil.Format("2006-01-02")
	}
	return d.Repeat
}

// SplitRRule is RRule backwards, for an event opened to edit: the end goes to
// its own fields and the rest stays the frequency.
func SplitRRule(rule string, d *Draft) {
	d.RepeatCount, d.RepeatUntil = 0, time.Time{}
	var keep []string
	for _, part := range strings.Split(rule, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToUpper(kv[0]) {
		case "COUNT":
			if n, err := strconv.Atoi(kv[1]); err == nil {
				d.RepeatCount = n
				continue
			}
		case "UNTIL":
			if day, err := time.Parse("2006-01-02", kv[1]); err == nil {
				d.RepeatUntil = day
				continue
			}
		}
		keep = append(keep, part)
	}
	d.Repeat = strings.Join(keep, ";")
}

// RepeatText reads the «✏️ Своя частота» answer: a period and, optionally, an
// end, and nothing else.
func RepeatText(text string, now time.Time) (Draft, bool) {
	p := ParseLine(text, now)
	if p.Draft.Repeat == "" || strings.TrimSpace(p.Title) != "" {
		return Draft{}, false
	}
	return p.Draft, true
}

// RepeatEndText reads the «Конец повтора» answer — «10 раз», «до 01.12», or a
// bare date — into d, replacing whichever end it had.
func RepeatEndText(text string, now time.Time, d *Draft) bool {
	toks := Tokenize(text)
	switch {
	case len(toks) == 2 && countWords[toks[1].Norm]:
		n, err := strconv.Atoi(toks[0].Norm)
		if err != nil || n <= 0 {
			return false
		}
		d.RepeatCount, d.RepeatUntil = n, time.Time{}
		return true
	case len(toks) == 2 && untilWords[toks[0].Norm]:
		toks = toks[1:]
		fallthrough
	case len(toks) == 1:
		day, ok := dateFromToken(toks[0].Norm, now)
		if !ok {
			return false
		}
		d.RepeatUntil, d.RepeatCount = day, 0
		return true
	}
	return false
}
