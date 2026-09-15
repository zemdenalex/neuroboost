package parse

import "time"

// Weekday names, in every form a person actually types.
//
// 🔴 Russian is inflected: nobody writes «в среда», they write «в среду». A
// nominative-only table would silently fail on the most natural phrasing and
// leave the word in the title, which looks like the bot ignoring you.
//
// Two-letter abbreviations (пн, вт, ср…) are here by Denis's decision of
// 15.09. They are safe only because matching happens per token: «вт» inside
// «втроём» is not a token, so it cannot match. The residual risk — a title
// that IS one of these words — is what the confirmation card is for.
var weekdayWords = map[string]time.Weekday{
	// Russian, nominative and accusative.
	"понедельник": time.Monday,
	"вторник":     time.Tuesday,
	"среда":       time.Wednesday,
	"среду":       time.Wednesday,
	"четверг":     time.Thursday,
	"пятница":     time.Friday,
	"пятницу":     time.Friday,
	"суббота":     time.Saturday,
	"субботу":     time.Saturday,
	"воскресенье": time.Sunday,
	"воскресение": time.Sunday,

	// Russian abbreviations.
	"пн": time.Monday,
	"вт": time.Tuesday,
	"ср": time.Wednesday,
	"чт": time.Thursday,
	"пт": time.Friday,
	"сб": time.Saturday,
	"вс": time.Sunday,

	// English, full and abbreviated.
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sunday":    time.Sunday,
	"mon":       time.Monday,
	"tue":       time.Tuesday,
	"tues":      time.Tuesday,
	"wed":       time.Wednesday,
	"thu":       time.Thursday,
	"thur":      time.Thursday,
	"thurs":     time.Thursday,
	"fri":       time.Friday,
	"sat":       time.Saturday,
	"sun":       time.Sunday,
}

// weekdayShift is how many whole weeks a modifier moves the nearest match.
//
// 🔴 Denis's rule, 15.09, and the pair of cases that pins it: on a Wednesday,
// «среда» is TODAY, and «следующая среда» is seven days out. So "next" is not
// "the one after the one you meant" — it is the nearest match plus a week, and
// the nearest match includes today.
var weekdayShift = map[string]int{
	"следующая": 1, "следующий": 1, "следующее": 1, "следующую": 1, "след": 1,
	"будущая": 1, "будущий": 1, "будущую": 1,
	"next": 1,

	"эта": 0, "этот": 0, "это": 0, "эту": 0,
	"this": 0,

	"прошлая": -1, "прошлый": -1, "прошлое": -1, "прошлую": -1,
	"last": -1, "past": -1,
}

// dayPrepositions are swallowed along with the day so they do not wash up in
// the title: «в среду оркестр» must be titled «оркестр», not «в оркестр».
var dayPrepositions = map[string]bool{"в": true, "во": true, "on": true}

// recogniseWeekday claims a weekday word, its modifier and its preposition,
// and writes the resulting date into the draft.
//
// Returns false and touches nothing when no weekday token is present. A
// modifier on its own is not a day: «следующий раз» is a title.
func recogniseWeekday(toks []Token, now time.Time, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		target, ok := weekdayWords[t.Norm]
		if !ok {
			continue
		}

		// Walk backwards over an optional modifier and an optional preposition.
		first := i
		shift := 0
		if j := first - 1; j >= 0 && toks[j].Field == FieldNone {
			if s, isMod := weekdayShift[toks[j].Norm]; isMod {
				shift = s
				first = j
			}
		}
		if j := first - 1; j >= 0 && toks[j].Field == FieldNone && dayPrepositions[toks[j].Norm] {
			first = j
		}

		d.Day = weekdayDate(target, shift, now)
		d.HasDay = true
		for k := first; k <= i; k++ {
			toks[k].Field = FieldDay
		}
		return true
	}
	return false
}

// weekdayDate turns a weekday plus a week shift into a date at local midnight.
func weekdayDate(target time.Weekday, shift int, now time.Time) time.Time {
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if shift < 0 {
		// The nearest one STRICTLY in the past: «прошлая среда» said on a
		// Wednesday means a week ago, not today.
		back := (int(base.Weekday()) - int(target) + 7) % 7
		if back == 0 {
			back = 7
		}
		return base.AddDate(0, 0, -back-7*(-shift-1))
	}

	ahead := (int(target) - int(base.Weekday()) + 7) % 7
	return base.AddDate(0, 0, ahead+7*shift)
}
