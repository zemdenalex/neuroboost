package parse

import (
	"strconv"
	"strings"
)

// Interval reads how long to wait, in the words people use for it.
//
// 🔴 It lives in `parse` rather than beside the screen that asks, because this
// is VOCABULARY — words the bot recognises in what the user types — not text it
// shows. The i18n scan draws exactly that line and refuses Russian literals in
// the rendering packages: «час» here is not a label to translate, it is a token
// that must keep working whichever interface language is set.

// Interval reads «40», «40 минут», «2 часа», «2h», «1.5 часа» is NOT read —
// a fraction is ambiguous between a decimal comma and a typo, and guessing an
// alarm time wrong is worse than asking again.
func Interval(text string) (int, bool) {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return 0, false
	}

	// 🔴 A leading minus is a refusal, not a character to skip over. Without
	// this, «-5» read as 5 and the reminder came back five minutes later — a
	// nonsense input answered with a confident action.
	if strings.HasPrefix(t, "-") {
		return 0, false
	}

	var digits strings.Builder
	for _, r := range t {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
			continue
		}
		if digits.Len() > 0 {
			break
		}
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil || n <= 0 {
		return 0, false
	}

	unit := 1
	switch {
	case strings.Contains(t, "час") || strings.Contains(t, "hour") || strings.HasSuffix(t, "ч") || strings.HasSuffix(t, "h"):
		unit = 60
	// ⚠ «день» contains neither «дн» nor «дня» — the stem changes. Matching on
	// «дн» alone read «1 день» as one MINUTE and answered confidently.
	case strings.Contains(t, "дн") || strings.Contains(t, "ден") ||
		strings.Contains(t, "day") || strings.HasSuffix(t, "д") || strings.HasSuffix(t, "d"):
		unit = 1440
	}
	total := n * unit

	// 🔴 The API caps a snooze at a day (reminders/action.go:32) and silently
	// clamps beyond it. Refusing here instead means the person is told, rather
	// than being promised three days and reminded tomorrow.
	if total > 24*60 {
		return 0, false
	}
	return total, true
}
