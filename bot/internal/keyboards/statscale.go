package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// StatsScaleName names a statistics scale (spec 22.09 §3) — the one wording
// for the 📏 button, ⚙️ Settings and onboarding.
func StatsScaleName(lang i18n.Lang, kind string) string {
	switch kind {
	case "work":
		return i18n.T(lang, "рабочие часы", "work hours")
	case "peak":
		return i18n.T(lang, "по максимуму", "busiest = full")
	default:
		return i18n.T(lang, "24 ч", "24 h")
	}
}

// StatsScale is the one screen for choosing the scale. Denis 22.09: «you can
// change to whatever in settings (and during onboarding it should ask too and
// button in statistics)» — three entrances, one screen, so they cannot drift.
// prefix is "scl_" from Settings or "ob_sc_" from onboarding; back is where
// the last button leads.
func StatsScale(lang i18n.Lang, current, prefix, backData, backLabel string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, kind := range []string{"day24", "work", "peak"} {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(tick(kind == current)+StatsScaleName(lang, kind), prefix+kind)))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(backLabel, backData)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
