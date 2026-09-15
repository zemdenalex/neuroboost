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

var (
	// One clock time: 14:00, 9:05, 14.00.
	clockRe = regexp.MustCompile(`^(\d{1,2})[:.](\d{2})$`)
	// A range written without spaces: 14:00-15:00, 14:00–15:30.
	clockRangeRe = regexp.MustCompile(`^(\d{1,2})[:.](\d{2})[-–—](\d{1,2})[:.](\d{2})$`)
	// A day-first date: 16.09 or 16.09.2027.
	dateTokenRe = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})(?:\.(\d{4}))?$`)
)

// dashSeparators are the tokens that can stand between two times when the
// range is written with spaces: "14:00 - 15:00".
var dashSeparators = map[string]bool{"-": true, "–": true, "—": true}

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
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		m := dateTokenRe.FindStringSubmatch(t.Norm)
		if m == nil {
			continue
		}
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		if month < 1 || month > 12 || day < 1 || day > 31 {
			continue
		}

		year := now.Year()
		if m[3] != "" {
			year, _ = strconv.Atoi(m[3])
		}
		candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, now.Location())
		// A bare day.month already gone by means next year: nobody schedules
		// into the past on purpose.
		if m[3] == "" && candidate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
			candidate = candidate.AddDate(1, 0, 0)
		}

		d.Day = candidate
		d.HasDay = true
		toks[i].Field = FieldDay
		return true
	}
	return false
}

// recogniseTimeRange claims a clock time, and an end time when one is written
// either as one token ("14:00-15:00") or as three ("14:00 - 15:00").
func recogniseTimeRange(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}

		if m := clockRangeRe.FindStringSubmatch(t.Norm); m != nil {
			start, okStart := clockOffset(m[1], m[2])
			end, okEnd := clockOffset(m[3], m[4])
			if !okStart || !okEnd {
				continue
			}
			setTime(d, start, end, true)
			toks[i].Field = FieldTime
			return true
		}

		m := clockRe.FindStringSubmatch(t.Norm)
		if m == nil {
			continue
		}
		start, ok := clockOffset(m[1], m[2])
		if !ok {
			continue
		}

		// "14:00 - 15:00" — a separator and a second clock, as separate tokens.
		if i+2 < len(toks) && dashSeparators[toks[i+1].Text] && toks[i+1].Field == FieldNone && toks[i+2].Field == FieldNone {
			if em := clockRe.FindStringSubmatch(toks[i+2].Norm); em != nil {
				if end, okEnd := clockOffset(em[1], em[2]); okEnd {
					setTime(d, start, end, true)
					toks[i].Field, toks[i+1].Field, toks[i+2].Field = FieldTime, FieldTime, FieldTime
					return true
				}
			}
		}

		setTime(d, start, 0, false)
		toks[i].Field = FieldTime
		return true
	}
	return false
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

func clockOffset(hh, mm string) (time.Duration, bool) {
	h, _ := strconv.Atoi(hh)
	m, _ := strconv.Atoi(mm)
	if h > 23 || m > 59 {
		return 0, false
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, true
}

func setTime(d *Draft, start, end time.Duration, hasEnd bool) {
	d.Start = start
	d.End = end
	d.HasTime = true
	d.HasEnd = hasEnd
}
