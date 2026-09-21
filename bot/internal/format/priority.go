package format

import "strconv"

// Priority styles (spec 21.09 §B). Настя, 18.09: «смайлики слишком из разных
// цветов, нет одного стиля визуально»; Denis 21.09: «1-3 в настройках, и
// спрашивать на онбординге». Circles stay the default.
const (
	StyleCircles = "circles"
	StyleDot     = "dot"
	StyleDash    = "dash"
)

// Priority is the ONE way a priority is drawn. Handlers never call
// PriorityEmoji directly (handlers/prio_scan_test.go): a screen that did would
// stay in the old style after the user changed it, and nobody would notice.
func Priority(style string, p int) string {
	switch style {
	case StyleDot:
		// Filled for the urgent two, hollow for the rest; the digit keeps the
		// order readable without colour. 1 = most urgent (gotcha 4).
		switch {
		case p == 0:
			return "·0"
		case p <= 2:
			return "●" + strconv.Itoa(p)
		default:
			return "○" + strconv.Itoa(p)
		}
	case StyleDash:
		return "—"
	default:
		return PriorityEmoji(p)
	}
}
