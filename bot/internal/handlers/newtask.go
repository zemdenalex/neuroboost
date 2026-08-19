package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// taskCardText renders what was understood — and nothing else.
//
// A card that shows "Срок: —" for a task with no due date trains the reader to
// skip the line. Absent fields are absent.
func taskCardText(r parse.TaskResult, tz string) string {
	title := r.Title
	if title == "" {
		title = "(без названия)"
	}
	text := "➕ <b>Новая задача</b>\n"
	if r.Priority != nil {
		text += format.PriorityEmoji(*r.Priority) + " "
	}
	text += fmt.Sprintf("<b>%s</b>\n", format.Escape(title))

	var meta []string
	if r.DueDate != nil {
		meta = append(meta, "📅 "+r.DueDate.Format("02.01"))
	}
	if r.EstimatedMinutes != nil {
		meta = append(meta, "⏱ "+format.Duration(*r.EstimatedMinutes))
	}
	if len(r.Tags) > 0 {
		meta = append(meta, "🏷 "+format.Escape(strings.Join(r.Tags, ", ")))
	}
	if len(meta) > 0 {
		text += strings.Join(meta, " · ") + "\n"
	}
	return text
}

// wizardOrder is the sequence of optional fields, coarsest first: a priority is
// a judgement, a date is a commitment, an estimate is a guess.
var wizardOrder = []string{"priority", "due", "estimate"}

// nextWizardStep returns the next field to ask about, skipping anything the
// typed line already answered, and "done" when nothing is left.
func nextWizardStep(current string, has map[string]bool) string {
	if current == "done" {
		return "done"
	}
	start := 0
	if current != "start" && current != "" {
		for i, f := range wizardOrder {
			if f == current {
				start = i + 1
				break
			}
		}
	}
	for i := start; i < len(wizardOrder); i++ {
		if !has[wizardOrder[i]] {
			return wizardOrder[i]
		}
	}
	return "done"
}

// wizardHas reports which wizard fields FlowData already carries — either
// from the typed line or from an earlier wizard step. It maps the wizard's
// "estimate" step onto the FlowData key handleTaskCardSave actually reads,
// "minutes".
func wizardHas(flowData map[string]any) map[string]bool {
	_, hasPriority := flowData["priority"]
	_, hasDue := flowData["due"]
	_, hasMinutes := flowData["minutes"]
	return map[string]bool{
		"priority": hasPriority,
		"due":      hasDue,
		"estimate": hasMinutes,
	}
}

// wizardStepText is what the wizard message says while asking about a field.
func wizardStepText(step string) string {
	switch step {
	case "priority":
		return "📝 <b>Подробнее</b>\n\nПриоритет? (можно пропустить)"
	case "due":
		return "📝 <b>Подробнее</b>\n\nКогда сделать? (можно пропустить)"
	case "estimate":
		return "📝 <b>Подробнее</b>\n\nСколько времени займёт? (можно пропустить)"
	}
	return "📝 <b>Подробнее</b>"
}

// wizardKeyboardFor is the keyboard for a given wizard step. Every one of
// these carries both escapes — see keyboards.wizardEscapes.
func wizardKeyboardFor(step string) tgbotapi.InlineKeyboardMarkup {
	switch step {
	case "priority":
		return keyboards.WizardPriority()
	case "due":
		return keyboards.WizardDue()
	case "estimate":
		return keyboards.WizardEstimate()
	}
	return keyboards.TaskCard()
}

// advanceWizard moves to the next unanswered step and renders it, or — once
// nothing is left to ask — saves the task exactly as handleTaskCardSave would
// from the plain card. The wizard never gates on anything: reaching "done"
// from any step, including the very first, must produce the same task the
// card's ✅ Создать would have.
func (h *Handler) advanceWizard(chatID int64, messageID int, current string) {
	us := h.store.GetOrCreate(chatID)
	next := nextWizardStep(current, wizardHas(us.FlowData))
	if next == "done" {
		us.FlowStep = "card"
		h.handleTaskCardSave(chatID, messageID)
		return
	}
	us.FlowStep = "wizard:" + next
	h.editOrSend(chatID, messageID, wizardStepText(next), wizardKeyboardFor(next))
}

// handleTaskWizardStart is nt_wizard — "📝 Подробнее" pressed under the card.
func (h *Handler) handleTaskWizardStart(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "card" {
		return
	}
	h.advanceWizard(chatID, messageID, "start")
}

// handleTaskWizardSkip is nt_skip — skip the current step only.
func (h *Handler) handleTaskWizardSkip(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || !strings.HasPrefix(us.FlowStep, "wizard:") {
		return
	}
	current := strings.TrimPrefix(us.FlowStep, "wizard:")
	h.advanceWizard(chatID, messageID, current)
}

// handleWizardPriority is nt_p_* — raw is one of "1".."5" or "0" (Buffer).
//
// 🔴 Priority is inverted (1 Emergency .. 5 If Possible) and 0 (Buffer) is a
// real, distinct choice — it is written into FlowData as the int 0, which
// handleTaskCardSave's `.(int)` type-assertion later reads as present, not as
// "unset". Do not special-case 0 here.
func (h *Handler) handleWizardPriority(chatID int64, messageID int, raw string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "wizard:priority" {
		return
	}
	p, err := strconv.Atoi(raw)
	if err != nil {
		return
	}
	us.FlowData["priority"] = p
	h.advanceWizard(chatID, messageID, "priority")
}

// handleWizardDue is nt_d_* — raw is a day offset from today: 0, 1, or 7.
func (h *Handler) handleWizardDue(chatID int64, messageID int, raw string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "wizard:due" {
		return
	}
	offset, err := strconv.Atoi(raw)
	if err != nil {
		return
	}
	due := time.Now().In(h.location()).AddDate(0, 0, offset)
	us.FlowData["due"] = due.Format(time.RFC3339)
	h.advanceWizard(chatID, messageID, "due")
}

// handleWizardEstimate is nt_e_* — raw is minutes: 15, 30, 60, or 120.
func (h *Handler) handleWizardEstimate(chatID int64, messageID int, raw string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "wizard:estimate" {
		return
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil {
		return
	}
	us.FlowData["minutes"] = minutes
	h.advanceWizard(chatID, messageID, "estimate")
}
