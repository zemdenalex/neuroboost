package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
)

// Onboarding — the first /start.
//
// 🔴 Why it exists (ref/feedback/bot-pervye-testery-2026-09-16-17.md): the first
// outside user could not find the language and decided there was none; the
// second has not opened the bot at all — «слишком много всего, что-то новое
// пугает». Denis explained by hand what the first minute should have.
//
// Two screens and a closing message, nothing more, because «lots of steps» was
// the other complaint: language, the clock, then «just write to me». The flag
// lives in the settings blob (bot.onboarded), not in memory — the bot is
// redeployed by hand, and a flag in UserState would re-onboard everyone after
// every deploy.

const onboardFlow = "onboard"

// needsOnboarding reads the flag. A failed read answers NO: showing a
// questionnaire to everyone whenever the API hiccups is worse than missing one
// person once.
func (h *Handler) needsOnboarding(chatID int64) bool {
	us := h.store.GetOrCreate(chatID)
	if us.Onboarded {
		return false
	}
	done, err := h.api.Onboarded(us.AuthToken)
	if err != nil {
		return false
	}
	if done {
		us.Onboarded = true
		return false
	}
	return true
}

// startOnboarding opens the language screen. telegramLang is from.language_code:
// it pre-picks the language only when the user has never chosen one.
func (h *Handler) startOnboarding(chatID int64, messageID int, telegramLang string) {
	us := h.store.GetOrCreate(chatID)
	if stored, err := h.api.BotLang(us.AuthToken); err == nil && stored == "" {
		// ⚠ «ru*» → Russian, EVERYTHING else → English. Mufid's Arabic lands on
		// English: the choice becomes visible on the first screen, but no
		// language is added.
		guess := i18n.EN
		if strings.HasPrefix(strings.ToLower(telegramLang), "ru") {
			guess = i18n.RU
		}
		us.SetLang(string(guess))
		// Saved at once, not only when a language is tapped: most people keep
		// the pre-selected one, and an unsaved guess is guessed again on the
		// next start instead of being the person's choice.
		if err := h.api.SetBotLang(us.AuthToken, string(guess)); err != nil {
			log.Printf("onboarding: could not save the guessed language: %s", logsafe.Redact(err))
		}
	}
	us.CurrentFlow, us.FlowStep, us.FlowData = onboardFlow, "lang", map[string]any{}
	h.showOnboardLang(chatID, messageID)
}

func (h *Handler) showOnboardLang(chatID int64, messageID int) {
	lang := h.lang(chatID)
	// Both languages in the heading: this is the one screen shown before anyone
	// knows which language the reader has.
	body := i18n.T(lang,
		"🧠 <b>NeuroBoost</b>\n\nВыбери язык · Choose language",
		"🧠 <b>NeuroBoost</b>\n\nChoose language · Выбери язык")
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(mark(lang == i18n.RU)+i18n.Name(i18n.RU), "ob_lang_ru"),
			tgbotapi.NewInlineKeyboardButtonData(mark(lang == i18n.EN)+i18n.Name(i18n.EN), "ob_lang_en"),
		},
		{tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Дальше →", "Next →"), "ob_tz")},
		{tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Пропустить", "Skip"), "ob_skip")},
	}
	h.editOrSend(chatID, messageID, body, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// clockOffsets are the zones the clock screen offers: the account's own and
// its neighbours, as hours east of UTC. Centred on the account (review 17.09):
// a fixed UTC+2…+12 row never marked anything for a user in New York.
func clockOffsets(currentHours int) []int {
	var out []int
	for h := currentHours - 3; h <= currentHours+7; h++ {
		if h >= -12 && h <= 14 {
			out = append(out, h)
		}
	}
	return out
}

// showOnboardTZ asks what time it is rather than which zone the user is in:
// everyone knows the first, few could name «Asia/Yekaterinburg».
func (h *Handler) showOnboardTZ(chatID int64, messageID int) {
	lang := h.lang(chatID)
	now := time.Now()
	current := h.location(chatID)
	_, currentOffset := now.In(current).Zone()

	rows := [][]tgbotapi.InlineKeyboardButton{}
	row := []tgbotapi.InlineKeyboardButton{}
	for _, hours := range clockOffsets(currentOffset / 3600) {
		loc := time.FixedZone("", hours*3600)
		label := mark(hours*3600 == currentOffset) + now.In(loc).Format("15:04")
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("ob_tzset_%d", hours)))
		if len(row) == 4 {
			rows = append(rows, row)
			row = nil
		}
	}
	row = append(row, tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "Другое…", "Other…"), "ob_tzother"))
	rows = append(rows, row)
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Верно, дальше →", "✅ Correct, next →"), "ob_prio"),
	})

	body := fmt.Sprintf(i18n.T(lang,
		"🕐 <b>Сколько у тебя сейчас времени?</b>\n\nСейчас стоит: <b>%s</b> (%s). Если не так, нажми своё время.",
		"🕐 <b>What time is it for you now?</b>\n\nSet now: <b>%s</b> (%s). If that is wrong, tap your time."),
		now.In(current).Format("15:04"), current.String())
	h.editOrSend(chatID, messageID, body, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// zoneForOffset turns hours east of UTC into a zone name.
//
// 🔴 Etc/GMT signs are INVERTED by POSIX: UTC+5 is «Etc/GMT-5». Getting this
// backwards moves every event ten hours and still looks right on the button.
func zoneForOffset(hours int) string {
	switch {
	case hours == 0:
		return "Etc/UTC"
	case hours > 0:
		return fmt.Sprintf("Etc/GMT-%d", hours)
	default:
		return fmt.Sprintf("Etc/GMT+%d", -hours)
	}
}

var utcOffsetRe = regexp.MustCompile(`(?i)^(?:utc|gmt)?\s*([+-]\d{1,2})$`)

// zoneFromText reads «Asia/Yekaterinburg», «UTC+5», «+5». Returns "" when it
// reads nothing — the caller asks again rather than guessing.
func zoneFromText(text string) string {
	text = strings.TrimSpace(text)
	if m := utcOffsetRe.FindStringSubmatch(text); m != nil {
		hours, err := strconv.Atoi(m[1])
		if err == nil && hours >= -12 && hours <= 14 {
			return zoneForOffset(hours)
		}
		return ""
	}
	if strings.Contains(text, "/") {
		if _, err := time.LoadLocation(text); err == nil {
			return text
		}
	}
	return ""
}

// setTimezone writes the zone, keeping a NAMED zone when the user picked the
// offset it already has: someone in Moscow pressing their own time should stay
// «Europe/Moscow», not become «Etc/GMT-3».
func (h *Handler) setTimezone(chatID int64, zone string) bool {
	us := h.store.GetOrCreate(chatID)
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return false
	}
	now := time.Now()
	_, want := now.In(loc).Zone()
	_, have := now.In(h.location(chatID)).Zone()
	if want == have {
		return true
	}
	if err := h.api.SetTimezone(us.AuthToken, zone); err != nil {
		return false
	}
	us.TZ, us.TZKnown = zone, true
	return true
}

// finishOnboarding writes the flag. A failed write is not shown: the next
// /start simply offers onboarding again, which is the honest consequence.
func (h *Handler) finishOnboarding(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetOnboarded(us.AuthToken); err == nil {
		us.Onboarded = true
	}
	if us.CurrentFlow == onboardFlow {
		h.store.ClearFlow(chatID)
	}
}

// onboardClosing is the last message: the one habit worth having — write to
// the bot without pressing anything — and buttons into the three places a new
// user goes next. Buttons, not prose naming the menu: a screen never sends
// people to look for a button (TestNoScreenPointsAtAReplyButton).
func (h *Handler) onboardClosing(chatID int64, messageID int) {
	lang := h.lang(chatID)
	h.finishOnboarding(chatID)

	// The reply keyboard arrives with the first message, because an edit of the
	// inline message cannot attach one.
	done := tgbotapi.NewMessage(chatID, i18n.T(lang, "✅ Готово.", "✅ Done."))
	done.ReplyMarkup = keyboards.MainMenu(lang)
	if messageID != 0 {
		h.editOrSend(chatID, messageID, i18n.T(lang, "🧠 NeuroBoost", "🧠 NeuroBoost"), keyboards.None())
	}
	h.send(chatID, done)

	h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
		`✍️ <b>Просто напиши мне, что нужно</b>, без кнопок:

<code>завтра в 15 стоматолог напомни за час</code>

Я спрошу, событие это или задача, покажу, что понял, и создам только после подтверждения.`,
		`✍️ <b>Just write to me what you need</b>, no buttons:

<code>tomorrow at 15 dentist remind in 1h</code>

I will ask whether it is an event or a task, show what I understood, and create it only once you confirm.`),
		keyboards.OnboardNext(lang))
}

// handleOnboardText answers text typed during onboarding.
func (h *Handler) handleOnboardText(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	if us.FlowStep != "tz_text" {
		// Text on a button screen: the user wants to write, not to tap. Let
		// them — onboarding ends, and the line becomes a quick add.
		h.finishOnboarding(chatID)
		h.handleQuickAdd(chatID, text)
		return
	}
	zone := zoneFromText(text)
	if zone == "" || !h.setTimezone(chatID, zone) {
		h.sendText(chatID, h.t(chatID,
			"Не понял пояс. Напиши как «UTC+5» или «Asia/Yekaterinburg».",
			"Could not read that zone. Write it like «UTC+5» or «Asia/Yekaterinburg»."))
		return
	}
	us.FlowStep = "tz"
	h.showOnboardTZ(chatID, 0)
}

// handleOnboardCallback answers ob_. Returns false for anything else.
func (h *Handler) handleOnboardCallback(chatID int64, messageID int, data string, from *tgbotapi.User) bool {
	if !strings.HasPrefix(data, "ob_") {
		return false
	}
	us := h.store.GetOrCreate(chatID)

	if data == "ob_start" {
		// «❓ Как пользоваться» — onboarding again, on demand.
		code := ""
		if from != nil {
			code = from.LanguageCode
		}
		h.startOnboarding(chatID, messageID, code)
		return true
	}
	if us.CurrentFlow != onboardFlow {
		// A button under an old onboarding message. Nothing to resume.
		h.handleMenu(chatID, messageID)
		return true
	}

	switch {
	case data == "ob_lang_ru" || data == "ob_lang_en":
		lang := strings.TrimPrefix(data, "ob_lang_")
		us.SetLang(lang)
		// Saved now, not at the end: a user who picks English and then skips
		// has still chosen English.
		_ = h.api.SetBotLang(us.AuthToken, lang)
		h.showOnboardLang(chatID, messageID)
	case data == "ob_tz":
		us.FlowStep = "tz"
		h.showOnboardTZ(chatID, messageID)
	case strings.HasPrefix(data, "ob_tzset_"):
		hours, err := strconv.Atoi(strings.TrimPrefix(data, "ob_tzset_"))
		if err != nil || !h.setTimezone(chatID, zoneForOffset(hours)) {
			h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить пояс.", "❌ Could not save the timezone."))
			return true
		}
		h.showOnboardTZ(chatID, messageID)
	case data == "ob_tzother":
		us.FlowStep = "tz_text"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Напиши свой пояс: «UTC+5» или «Asia/Yekaterinburg».",
			"Write your zone: «UTC+5» or «Asia/Yekaterinburg»."), keyboards.None())
	case data == "ob_prio":
		// Denis 21.09: «спрашивать на онбординге» — after language and zone.
		us.FlowStep = "prio"
		h.handlePriorityPick(chatID, messageID, "", "ob_pr_")
	case strings.HasPrefix(data, "ob_pr_"):
		h.handlePriorityPick(chatID, messageID, strings.TrimPrefix(data, "ob_pr_"), "ob_pr_")
	case data == "ob_scale":
		// Denis 22.09: onboarding asks for the statistics scale too.
		us.FlowStep = "scale"
		h.handleScalePick(chatID, messageID, "", true)
	case strings.HasPrefix(data, "ob_sc_"):
		h.handleScalePick(chatID, messageID, strings.TrimPrefix(data, "ob_sc_"), true)
	case data == "ob_dt":
		// Spec 2026-09-22 §11: after the scale, whether day tasks are on.
		us.FlowStep = "daytasks"
		h.editOrSend(chatID, messageID, h.dayTasksIntro(chatID), keyboards.DayOnboard(h.lang(chatID), "ob_dt_on", "ob_dt_off"))
	case data == "ob_dt_on":
		if err := h.setDayPref(chatID, "day_tasks_enabled", true); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return true
		}
		h.showOnboardTarget(chatID, messageID, h.dayPrefs(chatID).Target)
	case strings.HasPrefix(data, "ob_dtn_"):
		n, err := strconv.Atoi(strings.TrimPrefix(data, "ob_dtn_"))
		if err != nil || n < 3 || n > 7 {
			return true
		}
		if err := h.setDayPref(chatID, "day_tasks_target", n); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return true
		}
		h.showOnboardTarget(chatID, messageID, n)
	case data == "ob_dt_off":
		if err := h.setDayPref(chatID, "day_tasks_enabled", false); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return true
		}
		h.onboardClosing(chatID, messageID)
	case data == "ob_finish":
		h.onboardClosing(chatID, messageID)
	case data == "ob_skip":
		h.finishOnboarding(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID, "Хорошо, пропускаем.", "Okay, skipping."), keyboards.None())
		h.handleStart(chatID)
	default:
		return false
	}
	return true
}

// showOnboardTarget is N in onboarding, right after «✅ Включить».
func (h *Handler) showOnboardTarget(chatID int64, messageID int, current int) {
	h.editOrSend(chatID, messageID, h.t(chatID,
		"🎯 <b>Сколько дел брать на день?</b>\n\nЦвет дня считается от этого числа. Поменять можно в любой момент.",
		"🎯 <b>How many things a day?</b>\n\nThe day's colour is counted against this number. You can change it any time."),
		keyboards.DayTargetPick(h.lang(chatID), current, "ob_dtn_", "ob_finish", h.t(chatID, "Дальше →", "Next →")))
}
