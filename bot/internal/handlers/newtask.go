package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/parse"
)

// taskCardText renders the task card.
//
// ⚠ The comment that used to stand here said the opposite: «absent fields are
// absent», on the argument that «Срок: —» trains the reader to skip the line.
// Denis overturned it on 18.09 — a field that only appears once filled cannot
// teach that it exists, and not knowing which fields exist was the actual
// complaint. The comment is replaced rather than deleted so the reversal is
// visible: both positions are reasonable, and this one was chosen by the
// person using the product.
func taskCardText(lang i18n.Lang, r parse.TaskResult, tz, style string) string {
	return taskCardTextFull(lang, r, tz, "", "", style)
}

// taskCardTextFull writes the task card, naming EVERY characteristic.
//
// 🔴 Denis, 18.09, answering «карточка задачи тоже называет все поля» with a
// flat «Нет». The event card had just been taught this and the task card was a
// separate renderer one file away, still printing only what was filled — which
// is how the same defect ships twice in one release.
//
// Repeat is deliberately absent: tasks have no recurrence in the database at
// all (v0.4.11.4 adds it). Naming a field that cannot exist yet would be the
// opposite failure — promising a control that is not there.
func taskCardTextFull(lang i18n.Lang, r parse.TaskResult, tz, calendar, description, style string) string {
	title := r.Title
	if title == "" {
		title = i18n.T(lang, "(без названия)", "(untitled)")
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, "➕ <b>Новая задача</b>\n", "➕ <b>New task</b>\n"))
	fmt.Fprintf(&b, "💬 <b>%s</b>\n", format.Escape(title))

	due := ""
	if r.DueDate != nil {
		due = r.DueDate.Format("02.01")
	}
	b.WriteString(fieldLine("📅", i18n.T(lang, "Срок:", "Due:"), orNone(lang, due)))

	// 🔴 The line that was missing on 20.09. Every other characteristic was
	// named here, «нет» or not, per Denis's rule of 18.09 — except the one the
	// release was about, so the card could not teach that tasks repeat and the
	// bot could not be told to make one.
	switch {
	case r.Rrule != "":
		b.WriteString(fieldLine("🔁", i18n.T(lang, "Повтор:", "Repeat:"), format.Escape(freqName(lang, r.Rrule))))
	case r.RepeatAsked:
		// Same wording as the event card: a question about to be asked is not
		// an absence, and «нет» here would answer it silently.
		b.WriteString(i18n.T(lang,
			"⚠ повтор: частота не указана, спрошу\n",
			"⚠ repeats: no frequency given, I will ask\n"))
	default:
		b.WriteString(fieldLine("🔁", i18n.T(lang, "Повтор:", "Repeat:"), noneWord(lang)))
	}

	priority := ""
	if r.Priority != nil {
		priority = format.Priority(style, *r.Priority) + " " + priorityName(lang, *r.Priority)
	}
	b.WriteString(fieldLine("🎯", i18n.T(lang, "Приоритет:", "Priority:"), orNone(lang, priority)))

	estimate := ""
	if r.EstimatedMinutes != nil {
		estimate = format.Duration(*r.EstimatedMinutes)
	}
	b.WriteString(fieldLine("⏱", i18n.T(lang, "Оценка:", "Estimate:"), orNone(lang, estimate)))

	b.WriteString(fieldLine("🏷", i18n.T(lang, "Теги:", "Tags:"),
		orNone(lang, format.Escape(strings.Join(r.Tags, ", ")))))
	b.WriteString(fieldLine("📁", i18n.T(lang, "Календарь:", "Calendar:"),
		orNone(lang, format.Escape(calendar))))
	// Tasks take the account's reminder preset the same way events do, so the
	// honest word is «как обычно», not «нет».
	b.WriteString(fieldLine("🔔", i18n.T(lang, "Напоминания:", "Reminders:"),
		i18n.T(lang, "как обычно", "as usual")))
	b.WriteString(fieldLine("📄", i18n.T(lang, "Описание:", "Description:"),
		orNone(lang, format.Escape(shorten(description, descriptionOnCard)))))

	return b.String()
}

// priorityName says the priority in words. 🔴 The scale is inverted —
// 1 = Emergency, 5 = If Possible, 0 = Buffer (gotcha 4) — so a bare number on a
// card is read backwards by everyone who has not memorised that.
func priorityName(lang i18n.Lang, p int) string {
	switch p {
	case 0:
		return i18n.T(lang, "буфер", "buffer")
	case 1:
		return i18n.T(lang, "срочно", "emergency")
	case 2:
		return i18n.T(lang, "высокий", "high")
	case 3:
		return i18n.T(lang, "обычный", "normal")
	case 4:
		return i18n.T(lang, "низкий", "low")
	case 5:
		return i18n.T(lang, "если получится", "if possible")
	}
	return ""
}

// wizardOrder is the sequence of optional fields, coarsest first: a priority is
// a judgement, a date is a commitment, an estimate is a guess.
//
// «repeat» sits after the date on purpose: a series is anchored on the due date
// when there is one, so «когда» has to be settled before «как часто».
var wizardOrder = []string{"priority", "due", "repeat", "estimate"}

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
func wizardStepText(lang i18n.Lang, step string, flowData map[string]any, loc *time.Location, style string) string {
	switch step {
	case "priority":
		if p, ok := flowData["priority"].(int); ok {
			return fmt.Sprintf(i18n.T(lang, "📝 <b>Подробнее</b>\n\nПриоритет? Сейчас: %s %s (можно заменить или пропустить)", "📝 <b>More</b>\n\nPriority? Now: %s %s (replace it or skip)"),
				format.Priority(style, p), format.PriorityLabel(lang, p))
		}
		return i18n.T(lang, "📝 <b>Подробнее</b>\n\nПриоритет? (можно пропустить)", "📝 <b>More</b>\n\nPriority? (or skip)")
	case "due":
		if d, ok := flowData["due"].(string); ok && d != "" {
			if t, err := time.Parse(time.RFC3339, d); err == nil {
				return fmt.Sprintf(i18n.T(lang, "📝 <b>Подробнее</b>\n\nКогда сделать? Сейчас: %s (можно заменить или пропустить)", "📝 <b>More</b>\n\nWhen is it due? Now: %s (replace it or skip)"),
					t.In(loc).Format("02.01"))
			}
		}
		return i18n.T(lang, "📝 <b>Подробнее</b>\n\nКогда сделать? (можно пропустить)", "📝 <b>More</b>\n\nWhen is it due? (or skip)")
	case "repeat":
		hint := wizardHint(lang, "repeat")
		if rule, ok := flowData["rrule"].(string); ok && rule != "" {
			now := i18n.T(lang, "📝 <b>Подробнее</b>\n\nПовторять? Сейчас: ", "📝 <b>More</b>\n\nRepeat? Now: ")
			return now + format.Escape(freqName(lang, rule)) + hint
		}
		ask := i18n.T(lang, "📝 <b>Подробнее</b>\n\nПовторять? (можно пропустить)", "📝 <b>More</b>\n\nRepeat? (or skip)")
		return ask + hint
	case "estimate":
		if m, ok := flowData["minutes"].(int); ok {
			return fmt.Sprintf(i18n.T(lang, "📝 <b>Подробнее</b>\n\nСколько времени займёт? Сейчас: %s (можно заменить или пропустить)", "📝 <b>More</b>\n\nHow long will it take? Now: %s (replace it or skip)"),
				format.Duration(m)) + wizardHint(lang, "estimate")
		}
		return i18n.T(lang, "📝 <b>Подробнее</b>\n\nСколько времени займёт? (можно пропустить)", "📝 <b>More</b>\n\nHow long will it take? (or skip)") +
			wizardHint(lang, "estimate")
	}
	return i18n.T(lang, "📝 <b>Подробнее</b>", "📝 <b>More</b>")
}

// wizardHint tells the reader that this step also takes a typed answer.
//
// 🔴 Denis, 21.09, after finding out that «5м» works: *«не понятно обычному
// пользователю, что можно свой вариант; пускай пишет: нажми на кнопку или
// напиши свой вариант, например 5м, 2ч, 3д — то же самое с повторами»*. The
// parser has read those for weeks; only the screen was silent about it.
//
// 🔴 A hint ONLY where typing actually works — «estimate» and «repeat» are the
// two steps handleWizardText answers. Promising it on a step that ignores text
// would be the same defect as the card that said «спрошу» and did not:
// a printed claim with nothing behind it.
func wizardHint(lang i18n.Lang, step string) string {
	switch step {
	case "estimate":
		return i18n.T(lang,
			"\n\nНажми кнопку или напиши своё: «5м», «90м», «2ч».",
			"\n\nPress a button or type your own: «5m», «90m», «2h».")
	case "repeat":
		return i18n.T(lang,
			"\n\nНажми кнопку или напиши своё: «раз в 3 дня», «каждые 2 недели».",
			"\n\nPress a button or type your own: «every 3 days», «every 2 weeks».")
	}
	return ""
}

// wizardKeyboardFor is the keyboard for a given wizard step. Every one of
// these carries both escapes — see keyboards.wizardEscapes. Each also marks
// the button matching what is already known, when the known value lands on
// one of the quick choices offered.
func wizardKeyboardFor(lang i18n.Lang, step string, flowData map[string]any, loc *time.Location) tgbotapi.InlineKeyboardMarkup {
	switch step {
	case "priority":
		var current *int
		if p, ok := flowData["priority"].(int); ok {
			current = &p
		}
		return keyboards.WizardPriority(lang, current)
	case "due":
		return keyboards.WizardDue(lang, wizardDueOffset(flowData, loc))
	case "repeat":
		rule, _ := flowData["rrule"].(string)
		return keyboards.WizardRepeat(lang, rule)
	case "estimate":
		var current *int
		if m, ok := flowData["minutes"].(int); ok {
			current = &m
		}
		return keyboards.WizardEstimate(lang, current)
	}
	return keyboards.TaskCard(lang, false)
}

// advanceWizard moves to the next unanswered step and renders it, or — once
// nothing is left to ask — saves the task exactly as handleTaskCardSave would
// from the plain card. The wizard never gates on anything: reaching "done"
// from any step, including the very first, must produce the same task the
// card's ✅ Создать would have.
func (h *Handler) advanceWizard(chatID int64, messageID int, current string) {
	us := h.store.GetOrCreate(chatID)

	// 🔴 A question asked FROM the card goes back to the card.
	//
	// «✅ Создать» on a draft that said «спрошу» opens the repeat step — and
	// «повтор» sits third in wizardOrder, so advancing from it would walk the
	// user into «Сколько времени займёт?» next. They pressed create, not
	// «📝 Подробнее»; Denis's standing rule about this bot is «каждое
	// замечание — в сторону меньшего числа шагов», and that would be one more.
	//
	// The branch lives here rather than at the three call sites that end this
	// step — button, typed period, «пропустить» — because all three must obey
	// it and only one of them is obvious.
	if fromCard, _ := us.FlowData["repeat_from_card"].(bool); fromCard && current == "repeat" {
		delete(us.FlowData, "repeat_from_card")
		us.FlowStep = "card"
		h.handleTaskCardSave(chatID, messageID)
		return
	}

	next := nextWizardStep(current, wizardHas(us.FlowData))
	if next == "done" {
		us.FlowStep = "card"
		h.handleTaskCardSave(chatID, messageID)
		return
	}
	h.showWizardStep(chatID, messageID, next)
}

// showWizardStep puts one step on screen without deciding which comes next.
//
// Extracted so a step can be reached out of order: «повтор» has to be askable
// straight from the card, because the card promised to ask.
func (h *Handler) showWizardStep(chatID int64, messageID int, step string) {
	us := h.store.GetOrCreate(chatID)
	us.FlowStep = "wizard:" + step
	loc := h.location(chatID)
	h.editOrSend(chatID, messageID,
		wizardStepText(h.lang(chatID), step, us.FlowData, loc, h.priorityStyle(chatID)),
		wizardKeyboardFor(h.lang(chatID), step, us.FlowData, loc))
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
	due := time.Now().In(h.location(chatID)).AddDate(0, 0, offset)
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
