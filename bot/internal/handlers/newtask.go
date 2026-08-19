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

// nextWizardStep returns the next field to ask about, and "done" once every
// step has been visited.
//
// 🔴 It used to skip a field the typed line already answered. Spec, part 3:
// "a step whose value is already known shows it as the current value and
// lets you replace it" — skipping made "📝 Подробнее" on a fully-parsed line
// create the task with zero screens, indistinguishable from "✅ Создать". The
// `has` parameter is kept only so callers do not need to change; it is no
// longer consulted for skipping. What IS known still reaches the screen —
// through wizardStepText/wizardKeyboardFor, which read FlowData directly and
// mark the current value instead of hiding the step.
func nextWizardStep(current string, has map[string]bool) string {
	_ = has
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
	if start < len(wizardOrder) {
		return wizardOrder[start]
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

// wizardDueOffset reports which of the wizard's three quick due choices
// (0/1/7 days out) a known due date lands on, in the timezone the wizard asks
// in. "" means either nothing is known, or it is known but does not land on
// one of the three — wizardStepText still shows it, just not as a marked
// button.
func wizardDueOffset(flowData map[string]any, loc *time.Location) string {
	d, ok := flowData["due"].(string)
	if !ok || d == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return ""
	}
	dateStr := t.In(loc).Format("2006-01-02")
	now := time.Now().In(loc)
	for _, offset := range []int{0, 1, 7} {
		if now.AddDate(0, 0, offset).Format("2006-01-02") == dateStr {
			return strconv.Itoa(offset)
		}
	}
	return ""
}

// wizardStepText is what the wizard message says while asking about a field.
// Spec, part 3: a step whose value is already known shows it as the current
// value. That has to work even when the known value is not one of the
// keyboard's quick choices (an arbitrary parsed date, say), so the text
// carries it independently of what the keyboard can mark.
func wizardStepText(step string, flowData map[string]any, loc *time.Location) string {
	switch step {
	case "priority":
		if p, ok := flowData["priority"].(int); ok {
			return fmt.Sprintf("📝 <b>Подробнее</b>\n\nПриоритет? Сейчас: %s %s (можно заменить или пропустить)",
				format.PriorityEmoji(p), format.PriorityLabel(p))
		}
		return "📝 <b>Подробнее</b>\n\nПриоритет? (можно пропустить)"
	case "due":
		if d, ok := flowData["due"].(string); ok && d != "" {
			if t, err := time.Parse(time.RFC3339, d); err == nil {
				return fmt.Sprintf("📝 <b>Подробнее</b>\n\nКогда сделать? Сейчас: %s (можно заменить или пропустить)",
					t.In(loc).Format("02.01"))
			}
		}
		return "📝 <b>Подробнее</b>\n\nКогда сделать? (можно пропустить)"
	case "estimate":
		if m, ok := flowData["minutes"].(int); ok {
			return fmt.Sprintf("📝 <b>Подробнее</b>\n\nСколько времени займёт? Сейчас: %s (можно заменить или пропустить)",
				format.Duration(m))
		}
		return "📝 <b>Подробнее</b>\n\nСколько времени займёт? (можно пропустить)"
	}
	return "📝 <b>Подробнее</b>"
}

// wizardKeyboardFor is the keyboard for a given wizard step. Every one of
// these carries both escapes — see keyboards.wizardEscapes. Each also marks
// the button matching what is already known, when the known value lands on
// one of the quick choices offered.
func wizardKeyboardFor(step string, flowData map[string]any, loc *time.Location) tgbotapi.InlineKeyboardMarkup {
	switch step {
	case "priority":
		var current *int
		if p, ok := flowData["priority"].(int); ok {
			current = &p
		}
		return keyboards.WizardPriority(current)
	case "due":
		return keyboards.WizardDue(wizardDueOffset(flowData, loc))
	case "estimate":
		var current *int
		if m, ok := flowData["minutes"].(int); ok {
			current = &m
		}
		return keyboards.WizardEstimate(current)
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
	loc := h.location()
	h.editOrSend(chatID, messageID, wizardStepText(next, us.FlowData, loc), wizardKeyboardFor(next, us.FlowData, loc))
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
