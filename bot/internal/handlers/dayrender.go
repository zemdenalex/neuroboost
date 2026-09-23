package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// dayTitle is «📌 Задачи дня · ср 23.09». The weekday is our own, not Go's —
// Go's names are English whatever the chat's language.
func dayTitle(lang i18n.Lang, day time.Time) string {
	return fmt.Sprintf("📌 <b>%s</b> · %s %s",
		i18n.T(lang, "Задачи дня", "Day tasks"),
		weekdayShort(lang, (int(day.Weekday())+6)%7), day.Format("02.01"))
}

// renderDayScreen is the text of the day screen (spec 2026-09-22 §8). Pure:
// the handler fetches the day and, for a day not yet taken, the proposal.
//
// A taken day's tasks are the screen's buttons (keyboards.DayScreen), so the
// text carries only what the buttons cannot: the colour and the count.
func renderDayScreen(lang i18n.Lang, d api.Day, proposal []api.DayItem, day, today time.Time) string {
	var b strings.Builder
	b.WriteString(dayTitle(lang, day) + "\n")

	past := day.Before(today)
	switch {
	case d.Confirmed:
		fmt.Fprintf(&b, "%s %s\n", format.DayLevel(d.Level),
			fmt.Sprintf(i18n.T(lang, "%d из %d", "%d of %d"), d.Done, d.Target))
		if len(d.Items) == 0 {
			b.WriteString("\n" + i18n.T(lang, "В этом дне задач нет.", "No tasks in this day."))
		}
	case past:
		// Denis, 22.09: a day not taken is ⬛, «не выбрал = не сделал».
		b.WriteString("⬛ " + i18n.T(lang, "День не был взят.", "The day was not taken."))
	case len(proposal) == 0:
		b.WriteString("\n" + i18n.T(lang,
			"Взять нечего: открытых задач нет. Добавь задачу, и я её предложу.",
			"Nothing to take: there are no open tasks. Add one and I will offer it."))
	default:
		b.WriteString("\n" + i18n.T(lang, "День ещё не взят. Беру этот набор?", "The day is not taken yet. Take this set?") + "\n\n")
		for _, it := range proposal {
			b.WriteString("• " + format.Escape(it.Title) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
