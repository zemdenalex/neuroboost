package handlers

import (
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/release"
)

// Report a bug, suggest a feature, and see what changed.
//
// Denis, 17.09: the same two things the web has had since v0.4.9 (a
// FeedbackButton posting to /api/feedback), plus release notes. The client call
// already existed here — api.SubmitFeedback, written and then never wired to
// anything — so this is screens, not plumbing.

const feedbackFlowPrefix = "fb:"

// handleFeedbackCallback owns fb/fb_* and whatsnew*.
func (h *Handler) handleFeedbackCallback(chatID int64, messageID int, data string) bool {
	lang := h.lang(chatID)
	switch data {
	case "fb":
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"Что расскажешь?", "What would you like to tell us?"),
			keyboards.FeedbackKinds(lang))
	case "fb_bug", "fb_idea":
		kind := "bug"
		prompt := i18n.T(lang,
			"Опиши, что сломалось, одним сообщением. Я передам.",
			"Describe what broke in one message. I will pass it on.")
		if data == "fb_idea" {
			kind = "feature"
			prompt = i18n.T(lang,
				"Расскажи, чего не хватает, одним сообщением. Я передам.",
				"Tell us what is missing in one message. I will pass it on.")
		}
		h.store.GetOrCreate(chatID).CurrentFlow = feedbackFlowPrefix + kind
		h.sendHTMLWithKeyboard(chatID, prompt, keyboards.BackToMenu(lang))
	case "whatsnew":
		n := release.Latest()
		h.editOrSend(chatID, messageID, whatsNewText(lang, n), keyboards.WhatsNew(lang))
	case "whatsnew_all":
		var b strings.Builder
		for _, n := range release.All() {
			b.WriteString(whatsNewText(lang, n) + "\n\n")
		}
		h.editOrSend(chatID, messageID, strings.TrimSpace(b.String()), keyboards.BackToMenu(lang))
	default:
		return false
	}
	return true
}

func whatsNewText(lang i18n.Lang, n release.Note) string {
	body := n.RU
	if lang == i18n.EN {
		body = n.EN
	}
	return "🆕 <b>" + format.Escape(n.Version) + "</b>\n\n" + format.Escape(body)
}

// handleFeedbackText sends what the user wrote.
//
// 🔴 The title is the first line of the message, not a second question. Denis's
// rule from the four passes on 17.09 is that every correction goes toward FEWER
// steps — and asking «а теперь заголовок» is one more step for someone who is
// already annoyed enough to be reporting a bug.
func (h *Handler) handleFeedbackText(chatID int64, flow, text string) {
	lang := h.lang(chatID)
	kind := strings.TrimPrefix(flow, feedbackFlowPrefix)
	text = strings.TrimSpace(text)
	h.store.ClearFlow(chatID)

	if text == "" {
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"Пустое сообщение, не отправил.", "Empty message, nothing sent."),
			keyboards.BackToMenu(lang))
		return
	}

	if err := h.api.SubmitFeedback(h.store.GetOrCreate(chatID).AuthToken, api.CreateFeedbackReq{
		Type:        kind,
		Title:       feedbackTitle(text),
		Description: text,
	}); err != nil {
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"Не смог отправить. Попробуй ещё раз чуть позже.",
			"Could not send it. Try again in a moment."), keyboards.BackToMenu(lang))
		return
	}

	h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
		"Спасибо, записал. Посмотрим.", "Thank you, noted. We will look at it."),
		keyboards.BackToMenu(lang))
}

// feedbackTitle takes the first sentence or line, capped so the admin list
// stays readable.
func feedbackTitle(text string) string {
	title := text
	if i := strings.IndexAny(title, ".\n"); i > 0 {
		title = title[:i]
	}
	if r := []rune(strings.TrimSpace(title)); len(r) > 60 {
		title = string(r[:60])
	}
	return strings.TrimSpace(title)
}
