package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// StatsNav is the statistics screen's buttons (spec 22.09 §1; Denis 21.09:
// «кнопки двумя рядами — период × сущность, ← → листают»).
//
// The whole state of the screen rides in the button — st_<period>_<entity>_<offset>
// — so an old message stays usable and no flow is needed. period is w|m|y|a,
// entity a|e|t|r.
func StatsNav(lang i18n.Lang, period, entity string, offset int, scaleLabel string) tgbotapi.InlineKeyboardMarkup {
	data := func(p, e string, off int) string { return fmt.Sprintf("st_%s_%s_%d", p, e, off) }

	var periods []tgbotapi.InlineKeyboardButton
	if period != "a" {
		periods = append(periods, tgbotapi.NewInlineKeyboardButtonData("←", data(period, entity, offset-1)))
	}
	for _, p := range []string{"w", "m", "y", "a"} {
		// Switching the period starts from the current one: «three weeks
		// back» means nothing once the screen is a year.
		periods = append(periods, tgbotapi.NewInlineKeyboardButtonData(
			tick(p == period)+statsPeriodName(lang, p), data(p, entity, 0)))
	}
	if period != "a" {
		periods = append(periods, tgbotapi.NewInlineKeyboardButtonData("→", data(period, entity, offset+1)))
	}

	var entities []tgbotapi.InlineKeyboardButton
	for _, e := range []string{"a", "e", "t", "r"} {
		entities = append(entities, tgbotapi.NewInlineKeyboardButtonData(
			tick(e == entity)+statsEntityName(lang, e), data(period, e, offset)))
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		periods,
		entities,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📏 Шкала: ", "📏 Scale: ")+scaleLabel,
				fmt.Sprintf("stsc_%s_%s_%d", period, entity, offset)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu"),
		),
	)
}

// statsPeriodName and statsEntityName are written call by call rather than as
// a table: the scan that keeps untranslated text out of the bot reads i18n.T
// call sites, and a table of bare literals is invisible to it (it caught the
// first version of this file, 22.09).
func statsPeriodName(lang i18n.Lang, code string) string {
	switch code {
	case "m":
		return i18n.T(lang, "Месяц", "Month")
	case "y":
		return i18n.T(lang, "Год", "Year")
	case "a":
		return i18n.T(lang, "Всё", "All")
	default:
		return i18n.T(lang, "Неделя", "Week")
	}
}

func statsEntityName(lang i18n.Lang, code string) string {
	switch code {
	case "e":
		return i18n.T(lang, "События", "Events")
	case "t":
		return i18n.T(lang, "Задачи", "Tasks")
	case "r":
		return i18n.T(lang, "Рефлексии", "Reflections")
	default:
		return i18n.T(lang, "Всё", "All")
	}
}
