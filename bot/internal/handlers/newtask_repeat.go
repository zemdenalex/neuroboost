package handlers

import (
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// The repeat step of the new-task wizard, and typed answers to any step.
//
// 🔴 Both exist because of one walk through the v0.4.11.4 checklist, 20.09.
// Denis stopped at section 2: «не могу создать повторяющуюся задачу … он нигде
// не спрашивал про повтор, и слова повтор он тоже не понимает». The API had the
// occurrence table, the day states, the postponement and the nagging; the bot
// had no field, no word and no button for any of it.

// handleWizardRepeat is nt_r_* — raw is a key of keyboards.RepeatCodes.
func (h *Handler) handleWizardRepeat(chatID int64, messageID int, raw string) {
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_task" || us.FlowStep != "wizard:repeat" {
		return
	}
	rule, known := keyboards.RepeatCodes[raw]
	if !known {
		return
	}
	// «Не повторять» deletes the key rather than storing "": handleTaskCardSave
	// sends an rrule only when one is present, and a one-off task must put no
	// rrule key on the wire at all.
	if rule == "" {
		delete(us.FlowData, "rrule")
	} else {
		us.FlowData["rrule"] = rule
	}
	delete(us.FlowData, "repeat_asked")
	h.advanceWizard(chatID, messageID, "repeat")
}

// handleWizardText is a line typed while a wizard step is on screen.
//
// 🔴 Until 20.09 every wizard step answered «Здесь нужна кнопка». Denis, with
// the chat log to prove it: «Сколько времени займёт?» — «5м» — «Здесь нужна
// кнопка», and then: «почему здесь нет своего варианта?» The buttons offer four
// durations and five repeats, while the PARSER has read «5м» and «раз в 3 дня»
// for weeks. The custom option existed all along; the screen refused to hear it.
//
// Reports whether this step takes typed answers at all. A line it cannot read
// is NOT an error and does NOT discard the draft: the bot says what it can
// read, and the step stays where it was.
func (h *Handler) handleWizardText(chatID int64, text string) bool {
	us := h.store.GetOrCreate(chatID)
	step := strings.TrimPrefix(us.FlowStep, "wizard:")
	lang := h.lang(chatID)
	loc := h.location(chatID)
	now := time.Now().In(loc)

	switch step {
	case "estimate":
		// The task parser already knows every spelling of a duration: a line
		// that is ONLY a duration parses to an estimate and an empty title.
		r := parse.ParseTask(text, now)
		if r.EstimatedMinutes == nil || strings.TrimSpace(r.Title) != "" {
			h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
				"Не понял длительность. Напиши как «5м», «30м» или «1ч», или нажми кнопку.",
				"I could not read that. Write it like «5м», «30м» or «1ч», or press a button."),
				wizardKeyboardFor(lang, step, us.FlowData, loc))
			return true
		}
		us.FlowData["minutes"] = *r.EstimatedMinutes
		h.advanceWizard(chatID, 0, "estimate")
		return true

	case "repeat":
		d, ok := parse.RepeatText(text, now)
		if !ok {
			h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
				"Не понял период. Напиши как «раз в 3 дня» или «каждые 2 недели», или нажми кнопку.",
				"I could not read that. Write it like «every 3 days», or press a button."),
				wizardKeyboardFor(lang, step, us.FlowData, loc))
			return true
		}
		us.FlowData["rrule"] = d.RRule()
		delete(us.FlowData, "repeat_asked")
		h.advanceWizard(chatID, 0, "repeat")
		return true
	}
	return false
}
