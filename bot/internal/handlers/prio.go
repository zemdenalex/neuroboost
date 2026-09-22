package handlers

import (
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// The priority symbol (spec 21.09 §B). Настя, 18.09: «смайлики слишком из
// разных цветов, нет одного стиля»; Denis 21.09: «1-3 в настройках, и
// спрашивать на онбординге». Circles by default.

// priorityStyleRaw is what the user chose — "" when nothing was chosen yet,
// which is what the one-time question looks for.
func (h *Handler) priorityStyleRaw(chatID int64) string {
	us := h.store.GetOrCreate(chatID)
	if us.PriorityStyleKnown {
		return us.PriorityStyle
	}
	v, err := h.api.BotSetting(us.AuthToken, "priority_style")
	if err != nil {
		return ""
	}
	us.PriorityStyle, us.PriorityStyleKnown = v, true
	return v
}

// priorityStyle is the style to draw with.
func (h *Handler) priorityStyle(chatID int64) string {
	switch s := h.priorityStyleRaw(chatID); s {
	case format.StyleDot, format.StyleDash:
		return s
	default:
		return format.StyleCircles
	}
}

// prio draws a priority the way this chat chose. Every handler goes through
// here — prio_scan_test.go refuses a direct format.PriorityEmoji.
func (h *Handler) prio(chatID int64, p int) string {
	return format.Priority(h.priorityStyle(chatID), p)
}

var priorityStyles = map[string]bool{format.StyleCircles: true, format.StyleDot: true, format.StyleDash: true}

// handlePriorityPick is the one screen for the style, from three entrances:
// Settings (prs_), onboarding (ob_pr_) and the one-time question (prq_).
// style == "" only shows it; a known style is saved first.
func (h *Handler) handlePriorityPick(chatID int64, messageID int, style, prefix string) {
	us := h.store.GetOrCreate(chatID)
	if style != "" {
		if !priorityStyles[style] {
			return
		}
		if err := h.api.SetBotSetting(us.AuthToken, "priority_style", style); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return
		}
		us.PriorityStyle, us.PriorityStyleKnown = style, true
	}

	backData, backLabel := "settings_menu", h.t(chatID, "« Настройки", "« Settings")
	switch prefix {
	case "ob_pr_":
		backData, backLabel = "ob_scale", h.t(chatID, "Дальше →", "Next →")
	case "prq_":
		backData, backLabel = "main_menu", h.t(chatID, "« Меню", "« Menu")
	}

	// One line per language: the untranslated-text scan reads i18n.T call
	// sites line by line, and a Russian continuation line is invisible to it.
	text := h.t(chatID,
		"🔘 <b>Символ приоритета</b>\n\nКак отмечать срочность в списках:\n\n🔴 позвонить в банк\n🟡 купить хлеб — <i>кружки</i>\n\n●1 позвонить в банк\n○3 купить хлеб — <i>точки с цифрой</i>\n\n— позвонить в банк\n— купить хлеб — <i>только порядок</i>\n\nПоменять можно в любой момент.",
		"🔘 <b>Priority symbol</b>\n\nHow urgency is marked in lists:\n\n🔴 call the bank\n🟡 buy bread — <i>circles</i>\n\n●1 call the bank\n○3 buy bread — <i>dots with a digit</i>\n\n— call the bank\n— buy bread — <i>order only</i>\n\nChange it any time.")
	h.editOrSend(chatID, messageID, text,
		keyboards.PriorityStyle(h.lang(chatID), h.priorityStyle(chatID), prefix, backData, backLabel))
}

// askPriorityOnce is the one-time question for people who finished
// onboarding before the choice existed (spec §B2): shown once, remembered as
// asked even when they walk away without choosing. Returns true if it showed.
func (h *Handler) askPriorityOnce(chatID int64, messageID int) bool {
	us := h.store.GetOrCreate(chatID)
	if !us.Onboarded || h.priorityStyleRaw(chatID) != "" {
		return false
	}
	asked, err := h.api.BotSetting(us.AuthToken, "priority_style_asked")
	if err != nil || asked != "" {
		return false
	}
	if err := h.api.SetBotSetting(us.AuthToken, "priority_style_asked", "1"); err != nil {
		// Not remembered = would ask on every menu. Better not to ask now.
		return false
	}
	h.handlePriorityPick(chatID, messageID, "", "prq_")
	return true
}

// showPriorityUnderBroadcast is «🔘 Выбрать символ» under the 11.4 broadcast.
// The one screen of this bot that deliberately does NOT replace the message it
// was pressed on: that message is the release notes, and they stay readable.
// Choosing writes the key askPriorityOnce looks for, so the one-time question
// is not asked afterwards (spec 21.09 §B2).
func (h *Handler) showPriorityUnderBroadcast(chatID int64) {
	h.handlePriorityPick(chatID, 0, "", "prq_")
}
