package parse

import "strings"

// calendarOpeners introduce a calendar by name: «календарь работа».
var calendarOpeners = map[string]bool{"календарь": true, "календаре": true, "calendar": true}

// RecogniseCalendar claims a calendar name and reports which of `names` it was.
//
// 🔴 It lives outside the recogniser pipeline because it needs data the parser
// cannot have: the user's own calendars, fetched per request. The pipeline is a
// pure function of a line and a clock, and it stays that way.
//
// Two forms are accepted:
//
//	«календарь работа»  — the opener names the next token, whatever it is
//	«работа»            — a bare token that EXACTLY equals a calendar's name
//
// ⚠ Exact, never partial. A calendar called «Работа» would otherwise turn every
// «работа» in a title into a calendar change — and the title is the one thing
// the user definitely typed on purpose.
func RecogniseCalendar(toks []Token, names []string, d *Draft) (int, bool) {
	norm := make([]string, len(names))
	for i, n := range names {
		norm[i] = strings.ToLower(strings.TrimSpace(n))
	}

	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}

		if calendarOpeners[t.Norm] && i+1 < len(toks) && toks[i+1].Field == FieldNone {
			want := toks[i+1].Norm
			for j, n := range norm {
				if n == want {
					toks[i].Field, toks[i+1].Field = FieldCalendar, FieldCalendar
					d.Calendar = names[j]
					return j, true
				}
			}
			continue
		}

		for j, n := range norm {
			if n != "" && n == t.Norm {
				toks[i].Field = FieldCalendar
				d.Calendar = names[j]
				return j, true
			}
		}
	}
	return -1, false
}
