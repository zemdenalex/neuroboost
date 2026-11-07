package keyboards

import (
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/parse"
)

// «⏰ До срока» — remind regularly until the deadline (Лешик and Denis, 07.11).
//
// Two targets share the screens, told apart by the prefix:
//
//	rrt_…_<taskID>  a saved task: every press writes through the API
//	rrd_…           the event draft on screen: the press changes the draft,
//	                and the draft's save writes it
//
// The id goes LAST so a UUID never has to be split out of the middle.

// RuleCodes maps a frequency button to a rule. Minutes for an interval, days
// for a day rule (whose times are picked on the next screen).
var RuleCodes = map[string]struct{ Minutes, Days int }{
	"m5": {Minutes: 5}, "m15": {Minutes: 15}, "m30": {Minutes: 30},
	"h1": {Minutes: 60}, "h3": {Minutes: 180},
	"d1": {Days: 1}, "d2": {Days: 2}, "d7": {Days: 7},
}

// RuleTimeChoices are the times offered as toggles; a time written in the
// line («в 12 и 16») that is not among them is still shown and kept.
var RuleTimeChoices = []string{"08:00", "09:00", "10:00", "12:00", "14:00", "16:00", "18:00", "20:00", "21:00"}

func ruleCb(prefix, rest, id string) string {
	if id == "" {
		return prefix + rest
	}
	return prefix + rest + "_" + id
}

// RulePicker is the frequency screen.
func RulePicker(lang i18n.Lang, prefix, id, back string, current *parse.RemindRule) tgbotapi.InlineKeyboardMarkup {
	mark := func(on bool, s string) string {
		if on {
			return "✓ " + s
		}
		return s
	}
	is := func(code string) bool {
		c := RuleCodes[code]
		if current == nil {
			return false
		}
		return (c.Minutes > 0 && current.EveryMinutes == c.Minutes) || (c.Days > 0 && current.EveryDays == c.Days)
	}
	btn := func(code, label string) tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardButtonData(mark(is(code), label), ruleCb(prefix, "s_"+code, id))
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btn("m5", i18n.T(lang, "5 мин", "5 min")), btn("m15", i18n.T(lang, "15 мин", "15 min")), btn("m30", i18n.T(lang, "30 мин", "30 min"))),
		tgbotapi.NewInlineKeyboardRow(btn("h1", i18n.T(lang, "Каждый час", "Hourly")), btn("h3", i18n.T(lang, "Каждые 3 ч", "Every 3 h"))),
		tgbotapi.NewInlineKeyboardRow(btn("d1", i18n.T(lang, "Каждый день", "Every day")), btn("d2", i18n.T(lang, "Через день", "Every other day")), btn("d7", i18n.T(lang, "Раз в неделю", "Weekly"))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark(current == nil, i18n.T(lang, "Выключить", "Off")), ruleCb(prefix, "s_off", id))),
		tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpRule),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), back)),
	)
}

// RuleTimes is the second screen of a day rule: times as toggles, then save.
func RuleTimes(lang i18n.Lang, prefix, id string, days int, chosen []string) tgbotapi.InlineKeyboardMarkup {
	on := map[string]bool{}
	for _, t := range chosen {
		on[t] = true
	}
	choices := append([]string{}, RuleTimeChoices...)
	for _, t := range chosen {
		if !contains(choices, t) {
			choices = append(choices, t)
		}
	}
	d := strconv.Itoa(days)
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, t := range choices {
		label := t
		if on[t] {
			label = "✓ " + t
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, ruleCb(prefix, "t_"+d+"_"+strings.ReplaceAll(t, ":", ""), id)))
		if len(row) == 3 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpRule),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "💾 Готово", "💾 Done"), ruleCb(prefix, "ok_"+d, id))))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// RuleBack is the one-button way back from a screen that cannot go on.
func RuleBack(lang i18n.Lang, back string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(HelpButton(lang, HelpRule),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), back)))
}
