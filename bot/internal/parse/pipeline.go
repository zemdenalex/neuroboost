package parse

import "time"

// recogniser is one pass over the tokens. Every one of them marks the tokens it
// claims and writes its own field of the draft, and none of them touches the
// title — the title is whatever is left unclaimed at the end.
type recogniser struct {
	Name string
	Run  func(toks []Token, now time.Time, d *Draft) bool
}

// recognisers, in the order they run.
//
// 🔴 The order is part of the contract, not an implementation detail, and
// three places in it are load-bearing:
//
//   - date before time, because a dot is ambiguous: «16.09» is a date and
//     «14.00» is a time, and only trying the date first separates them;
//   - repeat before weekday, because «каждый вторник» is a rule AND a start
//     day, and the repeat pass deliberately leaves the weekday token behind;
//   - all-day after time, because «весь день» has to be able to drop a time
//     that was already read.
//
// TestRecogniserOrderIsFixed holds this list so a reordering during a refactor
// fails loudly instead of changing behaviour quietly.
var recognisers = []recogniser{
	{"date", func(t []Token, now time.Time, d *Draft) bool { return recogniseExplicitDate(t, now, d) }},
	{"relative-day", func(t []Token, now time.Time, d *Draft) bool { return recogniseRelativeDay(t, now, d) }},
	{"repeat", func(t []Token, _ time.Time, d *Draft) bool { return recogniseRepeat(t, d) }},
	{"weekday", func(t []Token, now time.Time, d *Draft) bool { return recogniseWeekday(t, now, d) }},
	{"time-range", func(t []Token, _ time.Time, d *Draft) bool { return recogniseTimeRange(t, d) }},
	{"time-word", func(t []Token, _ time.Time, d *Draft) bool { return recogniseTimeWord(t, d) }},
	{"all-day", func(t []Token, _ time.Time, d *Draft) bool { return recogniseAllDay(t, d) }},
	{"kind", func(t []Token, _ time.Time, d *Draft) bool { return recogniseKind(t, d) }},
	{"colour", func(t []Token, _ time.Time, d *Draft) bool { return recogniseColour(t, d) }},
	{"tags", func(t []Token, _ time.Time, d *Draft) bool { return recogniseTags(t, d) }},
}

// Parsed is a draft together with the tokens it came from.
//
// 🔴 The tokens are kept, not discarded, and that is what makes «изменить →
// название» implementable: the card can say which words it consumed and why,
// and a field can be replaced without re-running extraction over the
// replacement. The old parser returned only a result, so "stop honouring
// keywords" had nothing to switch off.
type Parsed struct {
	Draft  Draft
	Tokens []Token
	Title  string
}

// ParseLine reads one line into a draft.
//
// `now` is a parameter rather than a clock read so behaviour is testable and so
// the caller supplies the user's own timezone — the same contract the previous
// parser had, and worth keeping.
func ParseLine(line string, now time.Time) Parsed {
	toks := Tokenize(line)
	var d Draft
	for _, r := range recognisers {
		r.Run(toks, now, &d)
	}
	return Parsed{Draft: d, Tokens: toks, Title: Title(toks)}
}

// ParseLineRaw is the «изменить → название» path: no recogniser runs, and the
// text becomes the title exactly as typed.
//
// 🔴 This is a separate branch rather than a flag inside ParseLine. Denis,
// 15.09: «если выбирается изменить название то там уже слова не учитываются а
// текст напрямую в название переходит». A flag threaded through ten
// recognisers is a flag that one of them will one day ignore.
func ParseLineRaw(line string) string { return Title(Tokenize(line)) }
