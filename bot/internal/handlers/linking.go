package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
)

// Linking a Telegram chat and a website account, v0.4.11.5.
//
// Denis, 17.09: «dev website doesn't work with telegram login, I can't test
// it». The Telegram login widget is bound to the production domain and cannot
// be made to work on dev — that is not a bug to fix in code. So the bot hands
// out a link instead, and the direction works both ways.
//
// ⚠ The texts below are assembled from SHORT i18n.T calls rather than one long
// one. TestNoUntranslatedUserFacingText reads three lines above a Cyrillic
// literal looking for the T call, and a five-line argument puts its own tail
// outside that window — the scan then reports correct code, which is how a
// useful scan gets silenced. Short calls keep it honest.

// handleLinkingCallback owns lnk*.
func (h *Handler) handleLinkingCallback(chatID int64, messageID int, data string) bool {
	lang := h.lang(chatID)
	switch data {
	case "lnk":
		title := i18n.T(lang, "🔗 <b>Аккаунт на сайте</b>", "🔗 <b>Website account</b>")
		a := i18n.T(lang,
			"<b>Войти на сайт</b>: пришлю ссылку, она откроет сайт уже вошедшим.",
			"<b>Sign in</b>: I will send a link that opens the site already signed in.")
		b := i18n.T(lang,
			"<b>Привязать сайт</b>: пришлю код, его надо ввести в Профиле на сайте.",
			"<b>Link the website</b>: I will send a code to type into your Profile.")
		h.editOrSend(chatID, messageID, title+"\n\n"+a+"\n"+b, keyboards.Linking(lang))

	case "lnk_web":
		us := h.store.GetOrCreate(chatID)
		token, ttl, err := h.api.CreateLoginLink(us.AuthToken)
		if err != nil {
			h.editOrSend(chatID, messageID, h.linkingFailed(lang), keyboards.BackToMenu(lang))
			return true
		}
		head := i18n.T(lang, "🔗 Ссылка работает один раз и живёт ",
			"🔗 The link works once and lasts ")
		tail := i18n.T(lang,
			"Откроешь и попадёшь на сайт уже вошедшим. Задай там email и пароль.",
			"Opening it signs you in. Set an email and a password there.")
		// ⚠ A new message, not an edit: this is the thing the person has to tap
		// or copy, and an edited message scrolls away under whatever comes next.
		h.sendHTMLWithKeyboard(chatID,
			head+humanTTL(lang, ttl)+":\n\n"+
				format.Escape(h.webBase()+"/login/link?t="+token)+"\n\n"+tail,
			keyboards.BackToMenu(lang))

	case "lnk_code":
		us := h.store.GetOrCreate(chatID)
		code, ttl, err := h.api.CreateLinkCode(us.AuthToken)
		if err != nil {
			h.editOrSend(chatID, messageID, h.linkingFailed(lang), keyboards.BackToMenu(lang))
			return true
		}
		label := i18n.T(lang, "Код", "Code")
		where := i18n.T(lang,
			"Введи его на сайте: Профиль → Привязать Telegram. Код живёт ",
			"Type it on the site: Profile → Link Telegram. The code lasts ")
		ask := i18n.T(lang,
			"Потом я спрошу здесь, ты ли это.",
			"I will then ask you here whether it was you.")
		h.sendHTMLWithKeyboard(chatID,
			"🔢 "+label+": <b>"+format.Escape(code)+"</b>\n\n"+
				where+humanTTL(lang, ttl)+".\n"+ask,
			keyboards.BackToMenu(lang))

	default:
		return false
	}
	return true
}

func (h *Handler) linkingFailed(lang i18n.Lang) string {
	return i18n.T(lang,
		"⚠️ Не получилось. Попробуй через минуту.",
		"⚠️ That didn't work. Try again in a minute.")
}

// askAboutPersonalCalendars is the second question, asked only when both
// accounts own a personal calendar.
//
// The buttons hang off the SAME reminder id as the first question: one request,
// one notification, two answers. A second notification would be a second thing
// to deliver and a second thing to expire.
func (h *Handler) askAboutPersonalCalendars(chatID int64, msg *tgbotapi.Message, reminderID string) {
	lang := h.lang(chatID)
	merge := i18n.T(lang, "Слить в один", "Merge into one")
	both := i18n.T(lang, "Оставить оба", "Keep both")
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(merge, notifier.EncodeCallback("m", reminderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(both, notifier.EncodeCallback("b", reminderID)),
		),
	)

	head := i18n.T(lang, "📅 <b>Личные календари</b>", "📅 <b>Personal calendars</b>")
	body := i18n.T(lang,
		"У обоих аккаунтов есть личный календарь. Слить их или оставить оба?",
		"Both accounts have a personal calendar. Merge them, or keep both?")
	note := i18n.T(lang,
		"Второй станет обычным календарём.",
		"The second one becomes an ordinary calendar.")
	text := head + "\n\n" + body + "\n" + note

	// Edited in place when possible, so the thread stays one conversation
	// rather than a stack of questions.
	if msg != nil {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msg.MessageID, text, kb)
		edit.ParseMode = tgbotapi.ModeHTML
		if _, err := h.bot.Request(edit); err == nil {
			return
		}
	}
	h.sendHTMLWithKeyboard(chatID, text, kb)
}

// humanTTL says a number of seconds the way a person would.
func humanTTL(lang i18n.Lang, seconds int) string {
	minutes := seconds / 60
	if minutes < 1 {
		return i18n.T(lang, "меньше минуты", "less than a minute")
	}
	return notifier.HumanMinutes(lang, minutes)
}

// webBase is where the website lives, as far as this bot is concerned.
//
// ⚠ Taken from API_BASE, which for both deployed bots IS the site
// (https://dev.neuroboost.website and https://neuroboost.website — gotcha 19).
// Locally API_BASE is a docker address and the link will not open in a browser;
// that is a local-development fact, not a defect to code around, and inventing
// a second variable would be one more thing to set wrong on the server.
func (h *Handler) webBase() string {
	return strings.TrimSuffix(h.cfg.APIBase, "/")
}
