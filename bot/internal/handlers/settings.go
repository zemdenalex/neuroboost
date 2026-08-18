package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Settings in the bot, instead of a link to the web app.
//
// ⚙️ Settings used to be four lines pointing at https://neuroboost.website/
// settings — a button that answers "not here". For "usable from the phone
// without opening the laptop" that is the single worst shape a control can
// take: it costs a tap to learn nothing.
//
// Work hours first, because that is what v0.2.1 had (workhours_start_*,
// workhours_end_*, index.mjs:1103) and what the planner reads. The rest stays
// in the web for now, and the screen says so rather than implying completeness.

const (
	defaultWorkStart = "09:00"
	defaultWorkEnd   = "17:00"
)

// The hours offered. v0.2.1 gave three starts and three ends; these are wider
// because the cost of an extra button is a row, while the cost of a missing
// hour is that the feature does not apply to you at all.
var (
	startHours = []int{6, 7, 8, 9, 10, 11, 12}
	endHours   = []int{14, 15, 16, 17, 18, 19, 20, 21, 22}
)

func (h *Handler) handleSettings(chatID int64) {
	start, end := h.workHours(chatID)
	h.sendHTMLWithKeyboard(chatID, settingsText(start, end), keyboards.SettingsMenu())
}

func settingsText(start, end string) string {
	return "⚙️ <b>Настройки</b>\n\n" +
		fmt.Sprintf("🕘 Рабочие часы: <b>%s – %s</b>\n\n", start, end) +
		"Остальное — тема, масштаб, пресеты напоминаний — пока в вебе:\n" +
		"https://neuroboost.website/settings"
}

// workHours reads the two values, falling back to the same defaults the web
// shows. A read failure is reported as the defaults rather than as an error:
// this is a header line, and refusing to draw the whole settings screen because
// one label is unknown would be the worse trade.
func (h *Handler) workHours(chatID int64) (string, string) {
	us := h.store.GetOrCreate(chatID)
	settings, err := h.api.MySettings(us.AuthToken)
	if err != nil {
		return defaultWorkStart, defaultWorkEnd
	}
	return settingString(settings, "work_start", defaultWorkStart),
		settingString(settings, "work_end", defaultWorkEnd)
}

func settingString(settings map[string]any, key, fallback string) string {
	if v, ok := settings[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func (h *Handler) handleWorkHours(chatID int64) {
	start, end := h.workHours(chatID)
	h.sendHTMLWithKeyboard(chatID,
		fmt.Sprintf("🕘 <b>Рабочие часы</b>\n\nСейчас: <b>%s – %s</b>\n\nВыбери начало дня:", start, end),
		keyboards.WorkHoursStart(startHours))
}

// handleWorkHourSet writes one end of the range.
//
// Each tap saves immediately. v0.2.1 had a 💾 Save button, and it had no
// handler at all — the change was never written. An explicit save step in a
// chat is a place to lose work, not a safeguard.
func (h *Handler) handleWorkHourSet(chatID int64, data string) {
	which, hourStr, found := strings.Cut(data, "_")
	if !found || (which != "start" && which != "end") {
		return
	}
	hour, err := strconv.Atoi(hourStr)
	if err != nil || hour < 0 || hour > 23 {
		return
	}
	value := fmt.Sprintf("%02d:00", hour)

	current := defaultWorkStart
	other := defaultWorkEnd
	curStart, curEnd := h.workHours(chatID)
	if which == "start" {
		current, other = value, curEnd
	} else {
		current, other = curStart, value
	}

	// The web app accepts start ≥ end without complaint — a known gap noted in
	// WorkHoursSection.tsx. The bot refuses it: buttons cannot express "I meant
	// an overnight shift", so an inverted range here is always a mis-tap.
	if !rangeIsSane(current, other) {
		h.sendText(chatID, fmt.Sprintf(
			"⚠ %s – %s не получится: конец дня должен быть позже начала. Выбери другое.",
			current, other))
		return
	}

	us := h.store.GetOrCreate(chatID)
	if _, err := h.api.PatchSettings(us.AuthToken, map[string]any{"work_" + which: value}); err != nil {
		h.sendText(chatID, "❌ Не удалось сохранить: "+err.Error())
		return
	}

	if which == "start" {
		h.sendHTMLWithKeyboard(chatID,
			fmt.Sprintf("🕘 Начало: <b>%s</b>\n\nТеперь конец дня:", value),
			keyboards.WorkHoursEnd(endHours))
		return
	}
	h.sendHTMLWithKeyboard(chatID,
		fmt.Sprintf("✅ <b>Рабочие часы: %s – %s</b>", current, other),
		keyboards.BackToMenu())
}

// rangeIsSane compares "HH:MM" strings lexically, which is only valid because
// both sides are zero-padded 24-hour times produced here. Parsing would be
// heavier and no more correct for this input.
func rangeIsSane(start, end string) bool {
	return start < end
}
