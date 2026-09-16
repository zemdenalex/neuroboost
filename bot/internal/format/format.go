package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

func PriorityEmoji(p int) string {
	switch p {
	case 1:
		return "🔴"
	case 2:
		return "🟠"
	case 3:
		return "🟡"
	case 4:
		return "🟢"
	case 5:
		return "⚪"
	case 0:
		return "🔵"
	default:
		return "⚪"
	}
}

// PriorityLabel names what the number means. Priority is INVERTED here — 1 is
// Emergency, 5 is If Possible, 0 is the separate Buffer bucket — so a bare
// digit tells the reader nothing about which end is urgent. The word is the
// only cue.
//
// ⚠ These were English-only until 16.09, in a bot that spoke Russian
// everywhere else — a leftover from the first version nobody flagged because
// «Urgent» is legible to a Russian reader. Legible is not the same as
// translated.
func PriorityLabel(lang i18n.Lang, p int) string {
	switch p {
	case 1:
		return i18n.T(lang, "Срочно", "Emergency")
	case 2:
		return i18n.T(lang, "Важно", "Urgent")
	case 3:
		return i18n.T(lang, "Обычное", "Normal")
	case 4:
		return i18n.T(lang, "Неспешное", "Low")
	case 5:
		return i18n.T(lang, "Если получится", "If possible")
	case 0:
		return i18n.T(lang, "Буфер", "Buffer")
	default:
		return ""
	}
}

func FormatTime(isoTime string, tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}
	return t.In(loc).Format("15:04")
}

func FormatDate(isoTime string, tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}
	return t.In(loc).Format("Mon, Jan 2")
}

func Duration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	h := minutes / 60
	m := minutes % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

func Escape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
