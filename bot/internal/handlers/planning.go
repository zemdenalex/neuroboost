package handlers

import (
	"fmt"
	"sort"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Planning — the last capability the Go rewrite dropped from v0.2.1.
//
// v0.2.1's `plan_today` (index.mjs:266) printed today's events and the tasks at
// priority 2, and stopped there: a list you could read and not act on. There is
// a real endpoint now — GET /api/planning/week — which answers the question the
// screen is actually for: how much of the week is already committed, and what
// is still waiting to be placed.
//
// So this is not a transcription of the old screen. It shows the week's load,
// then the unplaced tasks as BUTTONS, each opening the scheduling flow added
// earlier tonight. Reading "you have 6 unscheduled tasks" on a phone and being
// unable to do anything about it is the shape this product keeps producing;
// two taps from here puts one in the calendar.

func (h *Handler) handlePlanning(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	plan, err := h.api.WeekPlan(us.AuthToken)
	if err != nil {
		h.editOrSend(chatID, messageID,
			"❌ Не удалось загрузить план: "+err.Error(), keyboards.BackToMenu())
		return
	}

	text := planningText(plan.ScheduledHours, plan.AvailableHours, len(plan.UnscheduledTasks))

	// Highest priority first — 1 is Emergency, 5 is If Possible. Sorting the
	// other way round is the standing trap in this codebase.
	tasks := append([]api.PlanningTask(nil), plan.UnscheduledTasks...)
	sort.SliceStable(tasks, func(i, j int) bool { return tasks[i].Priority < tasks[j].Priority })

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, t := range tasks {
		if i >= 8 {
			// Eight is what fits on a phone without the keyboard becoming a
			// scroll of its own. The count above already said how many there
			// are, so the cut is visible rather than silent.
			break
		}
		label := fmt.Sprintf("⏰ %s %s", format.PriorityEmoji(t.Priority), t.Title)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(truncateLabel(label), "task_sched_"+t.ID),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "planning"),
		tgbotapi.NewInlineKeyboardButtonData("« Menu", "main_menu"),
	))

	h.editOrSend(chatID, messageID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// planningText is the whole message, built from three numbers so it can be
// asserted without a bot, a store or a network.
func planningText(scheduled, available float64, unscheduled int) string {
	text := fmt.Sprintf("🗂 <b>План недели</b>\n\n%s\n", loadBar(scheduled, available))
	text += fmt.Sprintf("Занято <b>%s</b> из <b>%s</b>\n\n",
		formatHours(scheduled), formatHours(available))

	switch {
	case unscheduled == 0:
		text += "✨ Незапланированных задач нет."
	case unscheduled == 1:
		text += "Одна задача ждёт места в календаре — нажми, чтобы поставить:"
	default:
		text += fmt.Sprintf("Задач без времени: <b>%d</b>. Нажми любую, чтобы поставить в календарь:", unscheduled)
	}
	return text
}

// loadBar draws ten cells.
//
// The zero guard is load-bearing: available hours can legitimately be zero — a
// user who unticks every working day — and dividing by it prints NaN into the
// chat. Over 100% is possible and shows as a full bar with the true figure
// beside it.
//
// ⚠ There is deliberately no clamp on `filled`. One was written here, and a
// sabotage run found it could not fail: the render loop is bounded by `cells`,
// so a filled count of 15 already draws ten. A guard that cannot be reached is
// not protection, it is a claim that something was handled — and the next
// reader trusts it.
func loadBar(scheduled, available float64) string {
	const cells = 10
	if available <= 0 {
		return "▱▱▱▱▱▱▱▱▱▱ рабочие часы не заданы"
	}
	filled := int(scheduled / available * cells)
	bar := ""
	for i := 0; i < cells; i++ {
		if i < filled {
			bar += "▰"
		} else {
			bar += "▱"
		}
	}
	return fmt.Sprintf("%s %d%%", bar, int(scheduled/available*100))
}

// formatHours prints 6 rather than 6.0, and 6.5 rather than 6.50.
func formatHours(h float64) string {
	if h == float64(int(h)) {
		return fmt.Sprintf("%dч", int(h))
	}
	return fmt.Sprintf("%.1fч", h)
}

// truncateLabel keeps a button label inside what a phone can show.
//
// Counts RUNES, not bytes. Titles here are Russian, where every character is
// two bytes in UTF-8 — a byte-based cut halves the visible length and can slice
// a character in half, which Telegram renders as a replacement glyph.
func truncateLabel(s string) string {
	const max = 32
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
