package parse

import (
	"regexp"
	"strconv"
	"time"
)

// relativeDayWords maps a day word to an offset from today.
//
// ⚠ Longest-first ordering used to matter here because matching was done by
// substring — «послезавтра» had to be tried before «завтра» or it was read as
// «завтра» with a prefix. Matching per token removes that hazard entirely: a
// map lookup is exact, so the order of these entries carries no meaning.
var relativeDayWords = map[string]int{
	"сегодня":     0,
	"завтра":      1,
	"послезавтра": 2,
	"вчера":       -1,
	"позавчера":   -2,
	"today":       0,
	"tomorrow":    1,
	"yesterday":   -1,
}

// timeWords are the coarse times of day Denis named. The vocabulary is closed
// on purpose: "после обеда" and "часа в три" are guesses, and a guessed time
// written silently into a calendar is the defect this product already had.
var timeWords = map[string]time.Duration{
	"полдень":  12 * time.Hour,
	"полночь":  0,
	"утром":    9 * time.Hour,
	"утро":     9 * time.Hour,
	"вечером":  19 * time.Hour,
	"вечер":    19 * time.Hour,
	"ночью":    22 * time.Hour,
	"noon":     12 * time.Hour,
	"midnight": 0,
	"morning":  9 * time.Hour,
	"evening":  19 * time.Hour,
	"night":    22 * time.Hour,
}

// 🔴 Every shape below came out of one message Denis actually typed on 15.09,
// and four of the five were misread. Two of them moved half the time into the
// title: «12:00 -12:50 физра» was created as the event «12:50 физра».
//
// His rule: read them, and SAY that the reading was loose. A loose reading that
// does not announce itself is a guess wearing the clothes of a fact, and this
// product has shipped that mistake before under a different name.
var (
	// 14:00 · 14.00 · 13;00 — the separator says how sure we are, not whether
	// it parses.
	clockRe = regexp.MustCompile(`^(\d{1,2})([:.;])(\d{2})$`)
	// 14:00-15:00, and the halves may each lose their minutes: 10:40-12.
	clockRangeRe = regexp.MustCompile(`^(\d{1,2})(?:([:.;])(\d{2}))?[-–—](\d{1,2})(?:([:.;])(\d{2}))?$`)
	// «-12:50» and «-12» — a range whose dash stuck to the END, because the
	// space was typed before it and not after.
	endDashRe = regexp.MustCompile(`^[-–—](\d{1,2})(?:([:.;])(\d{2}))?$`)
	// «12:00-» — the mirror case, space after the dash.
	startDashRe = regexp.MustCompile(`^(\d{1,2})(?:([:.;])(\d{2}))?[-–—]$`)
	// A lone number. Only ever a time where a time is expected — see
	// timeExpectedAt.
	bareNumRe = regexp.MustCompile(`^(\d{1,2})$`)
	// A lone pair of digits, the minutes half of «10 00».
	bareMinRe = regexp.MustCompile(`^(\d{2})$`)

	// 16.09 or 16.09.2027, and 16/09 for the people whose keyboard puts the
	// slash where the dot should be.
	dateTokenRe = regexp.MustCompile(`^(\d{1,2})([./])(\d{1,2})(?:[./](\d{4}))?$`)
)

// dashSeparators are the tokens that can stand between two times when the
// range is written with spaces on both sides: "14:00 - 15:00".
var dashSeparators = map[string]bool{"-": true, "–": true, "—": true}

// A range written in words (Denis, 23.09): «с 01:00 до 22:00», «from 10 to
// 12». The date span (span.go) has run already and taken «с 14.10 до 16.10»,
// so what reaches these is a clock range or nothing.
var (
	rangeOpeners = map[string]bool{"с": true, "со": true, "from": true}
	rangeClosers = map[string]bool{"до": true, "to": true, "till": true, "until": true}
)

// clockOrBare reads «22:00» (strict) or «22» (loose) from one token.
func clockOrBare(norm string) (time.Duration, bool, bool) {
	if m := clockRe.FindStringSubmatch(norm); m != nil {
		v, ok := clockValue(m[1], m[3])
		return v, !strictClock(m[2], m[3]), ok
	}
	if m := bareNumRe.FindStringSubmatch(norm); m != nil {
		v, ok := clockValue(m[1], "")
		return v, true, ok
	}
	return 0, false, false
}

func recogniseRelativeDay(toks []Token, now time.Time, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		off, ok := relativeDayWords[t.Norm]
		if !ok {
			continue
		}
		base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		d.Day = base.AddDate(0, 0, off)
		d.HasDay = true
		toks[i].Field = FieldDay
		return true
	}
	return false
}

// recogniseExplicitDate claims a day-first date token.
//
// ⚠ It runs BEFORE the time recognisers, and that order is the only thing
// separating «16.09» (a date) from «16:09» (a time) when the writer used a dot
// for both. A token that cannot be a date — «09.30», there being no thirtieth
// month — is left alone and the time recogniser takes it.
func recogniseExplicitDate(toks []Token, now time.Time, d *Draft) bool {
	found := false
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		m := dateTokenRe.FindStringSubmatch(t.Norm)
		if m == nil {
			continue
		}
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[3])
		if month < 1 || month > 12 || day < 1 || day > 31 {
			continue
		}

		year := now.Year()
		if m[4] != "" {
			year, _ = strconv.Atoi(m[4])
		}
		candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, now.Location())
		// A bare day.month already gone by means next year: nobody schedules
		// into the past on purpose.
		if m[4] == "" && candidate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
			candidate = candidate.AddDate(1, 0, 0)
		}

		if d.HasDay {
			// A second date in the same line: kept, and the caller asks what
			// the pair means rather than guessing.
			d.MoreDays = append(d.MoreDays, candidate)
			toks[i].Field = FieldDay
			continue
		}
		d.Day = candidate
		d.HasDay = true
		if m[2] != "." {
			d.MarkUncertain(FieldDay)
		}
		toks[i].Field = FieldDay
		found = true
	}
	return found
}

// clockValue turns hours and optional minutes into an offset from midnight.
func clockValue(hh, mm string) (time.Duration, bool) {
	h, err := strconv.Atoi(hh)
	if err != nil || h > 23 {
		return 0, false
	}
	m := 0
	if mm != "" {
		m, err = strconv.Atoi(mm)
		if err != nil || m > 59 {
			return 0, false
		}
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, true
}

// strictClock reports whether a half of a time was written the one unambiguous
// way: a colon and two minutes.
func strictClock(sep, mm string) bool { return sep == ":" && mm != "" }

// compactClockRe is three or four digits with no separator.
var compactClockRe = regexp.MustCompile(`^(\d{1,2})(\d{2})$`)

// compactRangeRe is «1450-1800».
var compactRangeRe = regexp.MustCompile(`^(\d{3,4})[-–—](\d{3,4})$`)

// compactClock reads «1330», «900», «0100» as a clock time.
//
// 🔴 Three or four digits only: two digits are already a bare hour («12 обед»),
// and five are not a time at all. The hour and the minutes must both be real,
// so «2530» and «9999» fall through to the title — which is what makes
// «отжаться 1330 раз» safe, together with the caller's "time expected here".
func compactClock(norm string) (time.Duration, bool) {
	m := compactClockRe.FindStringSubmatch(norm)
	if m == nil || len(norm) < 3 {
		return 0, false
	}
	return clockValue(m[1], m[2])
}

// lastWordOf reports whether token i is the last unclaimed token of the line.
//
// 🔴 Denis, 17.09: «стоматолог 1500» — people write the time at the end as
// readily as at the start. Only a COMPACT clock (three or four digits) is read
// there: a bare «12» at the end is far more often a count, and «отжаться 1330
// раз» is safe because «раз» comes after it.
func lastWordOf(toks []Token, i int) bool {
	for j := i + 1; j < len(toks); j++ {
		if toks[j].Field == FieldNone {
			return false
		}
	}
	return i > 0
}

// timeExpectedAt reports whether a lone number at index i can be a start time.
//
// 🔴 Only where a time is expected: at the beginning of what is left of the
// entry, once the day has been taken. «12 обед» is noon; «отжаться 12 раз» is
// twelve repetitions, and reading that as noon would be worse than not reading
// «12 обед» at all. Denis asked for the first; the second is the price of
// asking, and this is where it is paid.
// timePrepositions introduce a clock time: «в 15», «at 3».
var timePrepositions = map[string]bool{"в": true, "во": true, "at": true}

// timePrepositionBefore reports whether token i follows an unclaimed «в»/«at».
//
// 🔴 Denis, 17.09: «завтра в 15 стоматолог». A bare number after «в» is a time
// wherever it stands — the preposition says so. It is still a LOOSE reading
// and is marked: «встреча в 2 этапа» is the price, and ⚠ on the card is how it
// gets paid.
func timePrepositionBefore(toks []Token, i int) bool {
	return i > 0 && toks[i-1].Field == FieldNone && timePrepositions[toks[i-1].Norm]
}

func timeExpectedAt(toks []Token, i int) bool {
	for j := 0; j < i; j++ {
		if toks[j].Field == FieldNone {
			return false
		}
	}
	return true
}

// recogniseTimeRange claims a clock time and, when one is written, an end.
//
// It accepts every shape in the regexps above and records whether the reading
// was strict. The caller shows the loose ones differently; it does not refuse
// them, because refusing is what produced «12:50 физра».
func recogniseTimeRange(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}

		// «1450-1800» — both halves written without a colon.
		if m := compactRangeRe.FindStringSubmatch(t.Norm); m != nil {
			start, okStart := compactClock(m[1])
			end, okEnd := compactClock(m[2])
			if okStart && okEnd {
				setTime(d, start, end, true)
				d.MarkUncertain(FieldTime)
				toks[i].Field = FieldTime
				return true
			}
		}

		// «14:00-15:00», «10:40-12», «10-12».
		if m := clockRangeRe.FindStringSubmatch(t.Norm); m != nil {
			start, okStart := clockValue(m[1], m[3])
			end, okEnd := clockValue(m[4], m[6])
			if okStart && okEnd {
				setTime(d, start, end, true)
				if !strictClock(m[2], m[3]) || !strictClock(m[5], m[6]) {
					d.MarkUncertain(FieldTime)
				}
				toks[i].Field = FieldTime
				return true
			}
			continue
		}

		// «с 01:00 до 22:00» — four tokens, both ends required. «с» alone is
		// not a time word: «встреча с 2 друзьями» has no «до» and stays a title.
		if rangeOpeners[t.Norm] && i+3 < len(toks) && rangeClosers[toks[i+2].Norm] &&
			toks[i+1].Field == FieldNone && toks[i+2].Field == FieldNone && toks[i+3].Field == FieldNone {
			start, startLoose, okStart := clockOrBare(toks[i+1].Norm)
			end, endLoose, okEnd := clockOrBare(toks[i+3].Norm)
			if okStart && okEnd {
				setTime(d, start, end, true)
				if startLoose || endLoose {
					d.MarkUncertain(FieldTime)
				}
				for k := i; k <= i+3; k++ {
					toks[k].Field = FieldTime
				}
				return true
			}
		}

		start, sep, mm, width, ok := readStart(toks, i)
		if !ok {
			continue
		}
		loose := !strictClock(sep, mm)

		last := i + width - 1
		claim := func(to int) {
			from := i
			// «в 15:00», «at 3» — the preposition goes with the time, or it
			// washes up in the title as «стоматолог в».
			if timePrepositionBefore(toks, i) {
				from = i - 1
			}
			for k := from; k <= to; k++ {
				toks[k].Field = FieldTime
			}
		}

		// An end in the following tokens: «- 15:00», «-15:00», «12:00-» then
		// «12:50», or a bare number after a dash.
		if end, endWidth, endLoose, hasEnd := readEnd(toks, last, sep == "-"); hasEnd {
			setTime(d, start, end, true)
			if loose || endLoose {
				d.MarkUncertain(FieldTime)
			}
			claim(last + endWidth)
			return true
		}

		setTime(d, start, 0, false)
		if loose {
			d.MarkUncertain(FieldTime)
		}
		claim(last)
		return true
	}
	return false
}

// readStart reads a start time beginning at token i, returning how many tokens
// it spans.
//
// sep is "-" when the start token carried a trailing dash, which tells the
// caller an end is coming even though no separator token follows.
func readStart(toks []Token, i int) (dur time.Duration, sep, mm string, width int, ok bool) {
	t := toks[i]

	if m := startDashRe.FindStringSubmatch(NormKeepingDash(t)); m != nil {
		if v, good := clockValue(m[1], m[3]); good {
			return v, "-", m[3], 1, true
		}
		return 0, "", "", 0, false
	}

	if m := clockRe.FindStringSubmatch(t.Norm); m != nil {
		if v, good := clockValue(m[1], m[3]); good {
			return v, m[2], m[3], 1, true
		}
		return 0, "", "", 0, false
	}

	// «1330», «900», «0100» — a clock with the colon left out. Denis, 17.09.
	// Only where a time is expected, and always LOOSE: «2026» is a year to
	// everyone except this branch.
	if v, ok := compactClock(t.Norm); ok && (timeExpectedAt(toks, i) || timePrepositionBefore(toks, i) || lastWordOf(toks, i)) {
		return v, "", "", 1, true
	}

	// A lone number, and possibly a lone pair of minutes after it: «10 00».
	if m := bareNumRe.FindStringSubmatch(t.Norm); m != nil && (timeExpectedAt(toks, i) || timePrepositionBefore(toks, i)) {
		if i+1 < len(toks) && toks[i+1].Field == FieldNone && bareMinRe.MatchString(toks[i+1].Norm) {
			if v, good := clockValue(m[1], toks[i+1].Norm); good {
				return v, " ", toks[i+1].Norm, 2, true
			}
		}
		if v, good := clockValue(m[1], ""); good {
			return v, "", "", 1, true
		}
	}
	return 0, "", "", 0, false
}

// readEnd reads the end of a range that starts after token `last`.
//
// dashTaken is true when the start token already carried the dash, in which
// case the next token is the end on its own.
func readEnd(toks []Token, last int, dashTaken bool) (dur time.Duration, width int, loose, ok bool) {
	next := last + 1
	if next >= len(toks) || toks[next].Field != FieldNone {
		return 0, 0, false, false
	}

	if dashTaken {
		if m := clockRe.FindStringSubmatch(toks[next].Norm); m != nil {
			if v, good := clockValue(m[1], m[3]); good {
				return v, 1, !strictClock(m[2], m[3]), true
			}
		}
		if m := bareNumRe.FindStringSubmatch(toks[next].Norm); m != nil {
			if v, good := clockValue(m[1], ""); good {
				return v, 1, true, true
			}
		}
		if v, ok := compactClock(toks[next].Norm); ok {
			return v, 1, true, true
		}
		return 0, 0, false, false
	}

	// «-12:50» — the dash stuck to the end because the space was typed before it.
	if m := endDashRe.FindStringSubmatch(NormKeepingDash(toks[next])); m != nil {
		if v, good := clockValue(m[1], m[3]); good {
			return v, 1, true, true
		}
		return 0, 0, false, false
	}

	// «15:00 до 16:00» — the end after a word instead of a dash.
	if rangeClosers[toks[next].Norm] && next+1 < len(toks) && toks[next+1].Field == FieldNone {
		if v, loose, good := clockOrBare(toks[next+1].Norm); good {
			return v, 2, loose, true
		}
	}

	// «- 15:00» — the dash stands alone.
	if dashSeparators[toks[next].Text] && next+1 < len(toks) && toks[next+1].Field == FieldNone {
		if m := clockRe.FindStringSubmatch(toks[next+1].Norm); m != nil {
			if v, good := clockValue(m[1], m[3]); good {
				return v, 2, !strictClock(m[2], m[3]), true
			}
		}
		if m := bareNumRe.FindStringSubmatch(toks[next+1].Norm); m != nil {
			if v, good := clockValue(m[1], ""); good {
				return v, 2, true, true
			}
		}
	}
	return 0, 0, false, false
}

// recogniseTimeWord claims a coarse time of day.
//
// 🔴 It claims the token even when a precise time has already been found. The
// word is a time word either way, so leaving it unclaimed would drop it into
// the title — «полдень 14:00 оркестр» would be titled «полдень оркестр».
func recogniseTimeWord(toks []Token, d *Draft) bool {
	found := false
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		off, ok := timeWords[t.Norm]
		if !ok {
			continue
		}
		toks[i].Field = FieldTime
		found = true
		if !d.HasTime {
			setTime(d, off, 0, false)
		}
	}
	return found
}

func setTime(d *Draft, start, end time.Duration, hasEnd bool) {
	d.Start = start
	d.End = end
	d.HasTime = true
	d.HasEnd = hasEnd
}
