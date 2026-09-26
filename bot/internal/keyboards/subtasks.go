package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// SubtaskItem is one subtask on its task's card.
type SubtaskItem struct {
	ID, Title string
	Done      bool
}

// SubtaskView is what a task card shows about subtasks (plan 23.09, day 2).
type SubtaskView struct {
	// ParentID is set when the card is itself a subtask: a way back up.
	ParentID string
	Items    []SubtaskItem
}

// subtaskLabel keeps a button short enough to read on a phone.
func subtaskLabel(mark, title string) string {
	r := []rune(title)
	if len(r) > 32 {
		title = string(r[:31]) + "…"
	}
	return mark + " " + title
}

// WithSubtasks adds the subtask rows to a task card, above its back row:
// ⬜ ticks an open subtask, ✅ opens a closed one, ➕ adds one, ↑ goes to the
// task a subtask belongs to. Every callback carries one id: 64 bytes hold one
// uuid with a prefix, never two.
func WithSubtasks(kb tgbotapi.InlineKeyboardMarkup, lang i18n.Lang, taskID string, v SubtaskView) tgbotapi.InlineKeyboardMarkup {
	rows := kb.InlineKeyboard
	if len(rows) == 0 {
		return kb
	}
	back := rows[len(rows)-1]
	out := append([][]tgbotapi.InlineKeyboardButton{}, rows[:len(rows)-1]...)
	for _, it := range v.Items {
		if it.Done {
			out = append(out, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(subtaskLabel("✅", it.Title), "task_action_"+it.ID)))
			continue
		}
		out = append(out, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(subtaskLabel("⬜", it.Title), "sb_d_"+it.ID)))
	}
	out = append(out, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Подзадача", "➕ Subtask"), "sb_add_"+taskID),
		HelpButton(lang, HelpSubtasks)))
	if v.ParentID != "" {
		out = append(out, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "↑ К задаче", "↑ To the task"), "task_action_"+v.ParentID)))
	}
	out = append(out, back)
	return tgbotapi.NewInlineKeyboardMarkup(out...)
}

// SubtaskPrompt is under «напиши подзадачу»: what this is, and cancel, which
// ends the question and shows the task again.
func SubtaskPrompt(lang i18n.Lang, parentID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		HelpButton(lang, HelpSubtasks),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "sb_x_"+parentID)))
}
