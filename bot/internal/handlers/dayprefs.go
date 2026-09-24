package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Day tasks can be switched off, and days before the first one taken are not
// coloured unless the person chose so (spec 2026-09-22 §11, Denis 24.09). Both
// are top-level settings, like day_tasks_target: the web will read them too.

// dayPrefs is what the day-tasks settings say.
type dayPrefs struct {
	On          bool
	PaintBefore bool
	Target      int
}

// readDayPrefs reads the settings blob. No key means on, not painting, 5.
func readDayPrefs(s map[string]any) dayPrefs {
	p := dayPrefs{On: true, Target: 5}
	if v, ok := s["day_tasks_enabled"].(bool); ok {
		p.On = v
	}
	if v, ok := s["day_tasks_paint_before"].(bool); ok {
		p.PaintBefore = v
	}
	if v, ok := s["day_tasks_target"].(float64); ok && v >= 3 && v <= 7 {
		p.Target = int(v)
	}
	return p
}

// dayPrefs reads the settings now. A failed read hides nothing: day tasks are
// taken as on, so a network blip does not make buttons vanish.
func (h *Handler) dayPrefs(chatID int64) dayPrefs {
	us := h.store.GetOrCreate(chatID)
	s, err := h.api.MySettings(us.AuthToken)
	if err != nil {
		return dayPrefs{On: true, Target: 5}
	}
	p := readDayPrefs(s)
	us.DayTasksOn, us.DayTasksKnown = p.On, true
	return p
}

// dayTasksOn is the cached switch, read once per chat.
func (h *Handler) dayTasksOn(chatID int64) bool {
	us := h.store.GetOrCreate(chatID)
	if us.DayTasksKnown {
		return us.DayTasksOn
	}
	return h.dayPrefs(chatID).On
}

// setDayPref writes one day-tasks setting through the read-merge-write
// (gotcha 21), and keeps the cached switch in step.
func (h *Handler) setDayPref(chatID int64, key string, v any) error {
	us := h.store.GetOrCreate(chatID)
	if _, err := h.api.PatchSettings(us.AuthToken, map[string]any{key: v}); err != nil {
		return err
	}
	if on, ok := v.(bool); ok && key == "day_tasks_enabled" {
		us.DayTasksOn, us.DayTasksKnown = on, true
	}
	return nil
}

// home is the home keyboard for this chat. Every handler goes through here
// (TestHandlersUseTheHomeMethod).
func (h *Handler) home(chatID int64) tgbotapi.InlineKeyboardMarkup {
	return keyboards.HomeInlineFor(h.lang(chatID), h.dayTasksOn(chatID))
}

// dayTasksIntro is the one explanation both entrances show: onboarding and
// the one-time question.
func (h *Handler) dayTasksIntro(chatID int64) string {
	return h.t(chatID,
		"📌 <b>Задачи дня</b>\n\nКаждое утро берёшь несколько дел на день. День красится по сделанному:\n🟩 всё · 🟨 почти · 🟧 больше половины · 🟥 мало · ⬛ ничего\n\nВ календаре это выглядит так: 🟩 21 ▅\n\nВключить? Поменять можно в любой момент.",
		"📌 <b>Day tasks</b>\n\nEach morning you take a few things for the day. The day is coloured by what got done:\n🟩 all · 🟨 nearly · 🟧 over half · 🟥 a little · ⬛ nothing\n\nIn the calendar it looks like this: 🟩 21 ▅\n\nSwitch on? You can change it any time.")
}

// askDayTasksOnce is the one-time question for people onboarded before D3
// (spec §11), shaped like askPriorityOnce: remembered as asked before it is
// shown, never shown to someone who has already chosen.
func (h *Handler) askDayTasksOnce(chatID int64, messageID int) bool {
	us := h.store.GetOrCreate(chatID)
	if !us.Onboarded || us.DayTasksAskDone {
		return false
	}
	s, err := h.api.MySettings(us.AuthToken)
	if err != nil {
		return false
	}
	if _, chosen := s["day_tasks_enabled"]; chosen {
		us.DayTasksAskDone = true
		return false
	}
	asked, err := h.api.BotSetting(us.AuthToken, "day_tasks_asked")
	if err != nil {
		return false
	}
	if asked != "" {
		us.DayTasksAskDone = true
		return false
	}
	if err := h.api.SetBotSetting(us.AuthToken, "day_tasks_asked", "1"); err != nil {
		// Not remembered = would ask on every menu. Better not to ask now.
		return false
	}
	us.DayTasksAskDone = true
	h.editOrSend(chatID, messageID, h.dayTasksIntro(chatID), keyboards.DayOnboard(h.lang(chatID), "dtq_on", "dtq_off"))
	return true
}
