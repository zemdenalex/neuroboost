package handlers

import "github.com/zemdenalex/neuroboost-bot/internal/i18n"

// The statistics screen lives in statsview.go (22.09). What is left here is
// the day names it shares with the calendar and «Сегодня».

// weekdayShort names a day of the ISO week, Monday first.
//
// ⚠ Built through i18n.T per day rather than as two package-level arrays: the
// scan that keeps untranslated text out of the bot reads call sites, and a table
// of bare literals is invisible to it — it would ship Russian to an English
// reader and nothing would complain.
func weekdayShort(lang i18n.Lang, i int) string {
	switch i {
	case 0:
		return i18n.T(lang, "пн", "Mo")
	case 1:
		return i18n.T(lang, "вт", "Tu")
	case 2:
		return i18n.T(lang, "ср", "We")
	case 3:
		return i18n.T(lang, "чт", "Th")
	case 4:
		return i18n.T(lang, "пт", "Fr")
	case 5:
		return i18n.T(lang, "сб", "Sa")
	default:
		return i18n.T(lang, "вс", "Su")
	}
}
