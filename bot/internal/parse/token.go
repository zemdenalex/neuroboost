package parse

import "strings"

// Field says which part of the draft consumed a token.
//
// FieldNone means nothing recognised the token, and that is precisely what
// makes it part of the title. The old parser worked the other way round — it
// cut recognised substrings OUT of the line and called the remainder a title —
// and that is why «10:00 завтрак» produced the title «к»: "завтра" was found
// inside "завтрак" and six letters were cut out of a word nobody had claimed.
type Field uint8

const (
	FieldNone Field = iota
	FieldDay
	FieldTime
	FieldRepeat
	FieldAllDay
	FieldKind
	FieldColour
	FieldCalendar
	FieldTag
	FieldReminder
)

// Token is one whitespace-delimited word of the input.
//
// 🔴 Text is what the user typed and is what rebuilds the title. Norm is the
// lowercased, edge-punctuation-stripped form and is the ONLY thing a recogniser
// compares against. Keeping the two apart is what lets «Оркестр,» be matched
// against a keyword list and still print with its comma.
type Token struct {
	Text  string
	Norm  string
	Field Field
}

// edgePunct is trimmed from both ends of Norm before matching. Only the ends:
// a colon inside 14:00 and a dash inside 14:00-15:00 carry meaning, and a
// comma inside «Иваном, срочно» belongs to the title.
const edgePunct = ` ,.;:!?()[]«»"'“”—–-`

// Tokenize splits a line into words, keeping each one verbatim.
//
// Word boundaries come free from the split, which is the whole point: a
// keyword either IS a token or is not present. No \b is used and none would
// help — \b is ASCII-minded and every keyword here may be Cyrillic.
func Tokenize(line string) []Token {
	fields := strings.Fields(line)
	toks := make([]Token, 0, len(fields))
	for _, f := range fields {
		toks = append(toks, Token{
			Text: f,
			Norm: strings.Trim(strings.ToLower(f), edgePunct),
		})
	}
	return toks
}

// Title joins everything no recogniser claimed, in the order it was typed.
func Title(toks []Token) string {
	parts := make([]string, 0, len(toks))
	for _, t := range toks {
		if t.Field == FieldNone {
			parts = append(parts, t.Text)
		}
	}
	return strings.Trim(strings.Join(parts, " "), edgePunct)
}

// dashEdgePunct is edgePunct WITHOUT the dashes.
//
// 🔴 Norm trims dashes off both ends, which is right for «оркестр —» and wrong
// for «-12:50». A leading dash there is not decoration: it is the separator of
// a range whose space was typed on the wrong side, and trimming it is what made
// «12:00 -12:50 физра» come out as the event «12:50 физра».
const dashEdgePunct = ` ,.;:!?()[]«»"'“”`

// NormKeepingDash is Norm with the leading and trailing dashes left in place.
func NormKeepingDash(t Token) string {
	return strings.Trim(strings.ToLower(t.Text), dashEdgePunct)
}
