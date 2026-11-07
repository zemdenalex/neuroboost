package handlers

import (
	"sort"
	"strconv"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/parse"
)

// «⏰ До срока» on a saved task (rrt_) and on the event draft (rrd_).
// Screens: keyboards/remindrule.go. Denis, 07.11: «напоминать каждый день в 12
// и 16 … или хотя бы в настройках задачи или события».
//
// Callback shapes (the id last, absent for the draft):
//
//	rr?_m[_id]                 the frequency screen
//	rr?_s_<code>[_id]          a frequency; a day rule goes on to the times
//	rr?_t_<days>_<HHMM>[_id]   toggle one time
//	rr?_ok_<days>[_id]         save the day rule with the chosen times

const ruleTimesKey = "rule_times"

func (h *Handler) handleRuleCallback(chatID int64, messageID int, data string) bool {
	var prefix, rest, taskID string
	switch {
	case strings.HasPrefix(data, "rrt_"):
		prefix, rest = "rrt_", strings.TrimPrefix(data, "rrt_")
		i := strings.LastIndex(rest, "_")
		if i < 0 {
			return true
		}
		rest, taskID = rest[:i], rest[i+1:]
	case strings.HasPrefix(data, "rrd_"):
		prefix, rest = "rrd_", strings.TrimPrefix(data, "rrd_")
	default:
		return false
	}
	parts := strings.Split(rest, "_")
	us := h.store.GetOrCreate(chatID)

	switch parts[0] {
	case "m":
		h.showRuleMenu(chatID, messageID, prefix, taskID)
	case "s":
		if len(parts) < 2 {
			return true
		}
		if parts[1] == "off" {
			h.saveRule(chatID, messageID, prefix, taskID, nil)
			return true
		}
		c, ok := keyboards.RuleCodes[parts[1]]
		if !ok {
			return true
		}
		if c.Minutes > 0 {
			h.saveRule(chatID, messageID, prefix, taskID, &parse.RemindRule{EveryMinutes: c.Minutes})
			return true
		}
		// A day rule: start from the times it already has, else the default.
		chosen := []string{parse.DefaultRuleTime}
		if cur := h.currentRule(chatID, prefix, taskID); cur != nil && len(cur.Times) > 0 {
			chosen = append([]string{}, cur.Times...)
		}
		us.FlowData[ruleTimesKey] = chosen
		h.editOrSend(chatID, messageID, h.t(chatID,
			"⏰ <b>До срока</b>\n\nВо сколько напоминать? Можно выбрать несколько.",
			"⏰ <b>Until due</b>\n\nAt what times? Pick one or several."),
			keyboards.RuleTimes(h.lang(chatID), prefix, taskID, c.Days, chosen))
	case "t":
		if len(parts) < 3 || len(parts[2]) != 4 {
			return true
		}
		days, _ := strconv.Atoi(parts[1])
		t := parts[2][:2] + ":" + parts[2][2:]
		chosen, _ := us.FlowData[ruleTimesKey].([]string)
		if i := indexOf(chosen, t); i >= 0 {
			chosen = append(chosen[:i], chosen[i+1:]...)
		} else {
			chosen = append(chosen, t)
		}
		sort.Strings(chosen)
		us.FlowData[ruleTimesKey] = chosen
		h.editOrSend(chatID, messageID, h.t(chatID,
			"⏰ <b>До срока</b>\n\nВо сколько напоминать? Можно выбрать несколько.",
			"⏰ <b>Until due</b>\n\nAt what times? Pick one or several."),
			keyboards.RuleTimes(h.lang(chatID), prefix, taskID, days, chosen))
	case "ok":
		if len(parts) < 2 {
			return true
		}
		days, _ := strconv.Atoi(parts[1])
		chosen, _ := us.FlowData[ruleTimesKey].([]string)
		delete(us.FlowData, ruleTimesKey)
		if len(chosen) == 0 {
			chosen = []string{parse.DefaultRuleTime}
		}
		h.saveRule(chatID, messageID, prefix, taskID, &parse.RemindRule{EveryDays: days, Times: chosen})
	}
	return true
}

func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

// currentRule is what the screen starts from.
func (h *Handler) currentRule(chatID int64, prefix, taskID string) *parse.RemindRule {
	if prefix == "rrd_" {
		if st, ok := draftOf(h, chatID); ok {
			return st.D.RemindRule
		}
		return nil
	}
	us := h.store.GetOrCreate(chatID)
	r, _ := h.api.RemindRule(us.AuthToken, "tasks", taskID)
	return r
}

func (h *Handler) showRuleMenu(chatID int64, messageID int, prefix, taskID string) {
	back := "dr_back"
	if prefix == "rrt_" {
		back = "task_action_" + taskID
		// A rule runs up to the due date; without one there is nothing to
		// run up to, and the API would refuse.
		if t, ok := h.taskByID(chatID, taskID); ok && t.DueDate == "" {
			h.editOrSend(chatID, messageID, h.t(chatID,
				"⏰ Сначала поставь задаче срок (📅 Срок): напоминания идут до него.",
				"⏰ Give the task a due date first (📅 Due): the reminders run up to it."),
				keyboards.RuleBack(h.lang(chatID), back))
			return
		}
	}
	cur := h.currentRule(chatID, prefix, taskID)
	text := h.t(chatID,
		"⏰ <b>До срока</b>\n\nКак часто напоминать, пока не наступит срок?",
		"⏰ <b>Until due</b>\n\nHow often should I remind you until it is due?")
	if cur != nil {
		text += "\n\n" + h.t(chatID, "Сейчас: ", "Now: ") + cur.Describe(h.lang(chatID) == i18n.RU)
	}
	h.editOrSend(chatID, messageID, text, keyboards.RulePicker(h.lang(chatID), prefix, taskID, back, cur))
}

// saveRule writes the rule: through the API for a task, into the draft for an
// event (its save sends it — draftflow createOne / eventedit).
func (h *Handler) saveRule(chatID int64, messageID int, prefix, taskID string, rule *parse.RemindRule) {
	if prefix == "rrd_" {
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		st.D.RemindRule = rule
		st.RuleTouched = true
		h.showCard(chatID, messageID)
		return
	}
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetRemindRule(us.AuthToken, "tasks", taskID, rule); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	if rule == nil {
		h.sendText(chatID, h.t(chatID, "⏰ Регулярные напоминания выключены.", "⏰ Regular reminders are off."))
	} else {
		h.sendText(chatID, "⏰ "+h.t(chatID, "Буду напоминать ", "I'll remind you ")+rule.Describe(h.lang(chatID) == i18n.RU)+".")
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

// syncEventRule sends the draft's rule after the event itself is saved. A
// rule failure does not undo the event — it is said, so it can be set again.
func (h *Handler) syncEventRule(chatID int64, eventID string, st draftState) string {
	if st.D.RemindRule == nil && !st.RuleTouched {
		return ""
	}
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetRemindRule(us.AuthToken, "events", eventID, st.D.RemindRule); err != nil {
		return h.t(chatID, "\n⚠ Регулярные напоминания не сохранились: ", "\n⚠ The regular reminders were not saved: ") + h.errorText(chatID, err)
	}
	return ""
}

// attachTaskRule sends a rule typed in the line to the task just created and
// returns a line for the receipt. A rule needs a due date: without one it is
// not stored, and the receipt says how to set it.
func (h *Handler) attachTaskRule(chatID int64, taskID string, rule *parse.RemindRule, hasDue bool) string {
	if rule == nil {
		return ""
	}
	if !hasDue {
		return "\n" + h.t(chatID,
			"⚠ «Напоминать до срока» нужен срок: поставь 📅 Срок и включи ⏰ До срока в задаче.",
			"⚠ «Remind until due» needs a due date: set 📅 Due, then ⏰ Until due on the task.")
	}
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetRemindRule(us.AuthToken, "tasks", taskID, rule); err != nil {
		return "\n" + h.t(chatID, "⚠ Регулярные напоминания не сохранились: ", "⚠ The regular reminders were not saved: ") + h.errorText(chatID, err)
	}
	return "\n⏰ " + h.t(chatID, "Буду напоминать ", "I'll remind you ") + rule.Describe(h.lang(chatID) == i18n.RU)
}

// ruleLine is the card's line for a rule, and for a date that does not exist.
func ruleLine(lang i18n.Lang, rule *parse.RemindRule, badDate string) string {
	var b strings.Builder
	if badDate != "" {
		b.WriteString(i18n.T(lang, "⚠ «"+badDate+"»: такой даты нет, выбери день", "⚠ «"+badDate+"» is not a real date, pick the day") + "\n")
	}
	if rule != nil {
		b.WriteString(fieldLine("⏰", i18n.T(lang, "До срока:", "Until due:"), rule.Describe(lang == i18n.RU)))
	}
	return b.String()
}
