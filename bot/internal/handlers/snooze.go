package handlers

import (
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// «Отложить на своё время».
//
// Denis, 18.09, after pressing the two fixed buttons: «и где свой вариант?»
// Ten minutes and an hour cover the common cases and neither covers "until
// after the meeting".
//
// The chat asks, because an arbitrary interval cannot be a button. A handful of
// common answers stay buttons anyway — most of the time the custom answer is
// one of four numbers, and making those cost a typed message would be a step
// backwards from the two buttons it replaces.

const snoozeFlowPrefix = "snz:"

// snoozeChoices are the intervals offered after «Своё», in minutes.
var snoozeChoices = []int{15, 30, 120, 180}

func (h *Handler) askSnoozeInterval(chatID int64, reminderID string) {
	lang := h.lang(chatID)
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = snoozeFlowPrefix + reminderID
	us.FlowStep = "minutes"

	var row []tgbotapi.InlineKeyboardButton
	for _, m := range snoozeChoices {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			"⏰ "+notifier.HumanMinutes(lang, m), "snz_"+strconv.Itoa(m)))
	}
	kb := tgbotapi.NewInlineKeyboardMarkup(
		row,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "snz_no")))

	h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
		"Через сколько напомнить? Нажми или напиши — «40 минут», «2 часа», «90».",
		"When should I remind you? Tap one, or write — «40 minutes», «2 hours», «90»."), kb)
}

// handleSnoozeCallback owns snz_*.
func (h *Handler) handleSnoozeCallback(chatID int64, messageID int, data string) bool {
	if !strings.HasPrefix(data, "snz_") {
		return false
	}
	us := h.store.GetOrCreate(chatID)
	reminderID := strings.TrimPrefix(us.CurrentFlow, snoozeFlowPrefix)
	h.store.ClearFlow(chatID)

	if data == "snz_no" {
		h.editOrSend(chatID, messageID, h.t(chatID, "Хорошо, не откладываю.", "All right, not postponing."), keyboards.None())
		return true
	}
	minutes, err := strconv.Atoi(strings.TrimPrefix(data, "snz_"))
	if err != nil {
		return true
	}
	h.applySnooze(chatID, reminderID, minutes)
	return true
}

// handleSnoozeText reads a typed interval.
func (h *Handler) handleSnoozeText(chatID int64, flow, text string) {
	reminderID := strings.TrimPrefix(flow, snoozeFlowPrefix)
	h.store.ClearFlow(chatID)

	minutes, ok := parse.Interval(text)
	if !ok {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID,
			"Не понял, через сколько. Напиши числом — «40», «2 часа».",
			"I did not get the interval. Write a number — «40», «2 hours»."), keyboards.None())
		return
	}
	h.applySnooze(chatID, reminderID, minutes)
}

func (h *Handler) applySnooze(chatID int64, reminderID string, minutes int) {
	lang := h.lang(chatID)
	if h.cfg.ServiceToken == "" {
		return
	}
	if err := h.api.NotificationAction(h.cfg.ServiceToken, chatID, reminderID, notifier.ActionSnooze, minutes); err != nil {
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"⚠️ Не получилось отложить — попробуй ещё раз.",
			"⚠️ Could not postpone it — try again."), keyboards.None())
		return
	}
	h.sendText(chatID, notifier.ActionReply(lang, notifier.ActionSnooze, minutes))
}
