package handlers

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// langTTL is how long a chat trusts a cached per-user setting (language,
// priority symbol) before asking the API again: short enough that a change
// made elsewhere (the priority symbol is also set in the web) reaches the chat
// within minutes, long enough that a burst of keypresses costs one read.
const langTTL = 5 * time.Minute

// lang returns the interface language for this chat.
//
// 🔴 A failed read is NOT cached. Caching it would lock the chat into Russian
// for the lifetime of the process because the API hiccuped once — and the bot
// is redeployed by hand, so "the lifetime of the process" can be weeks. A
// failure falls back for this one message and tries again on the next.
func (h *Handler) lang(chatID int64) i18n.Lang {
	us := h.store.GetOrCreate(chatID)
	if us.LangKnown && time.Since(us.LangAt) < langTTL {
		return i18n.Parse(us.Lang)
	}
	stored, err := h.api.BotLang(us.AuthToken)
	if err != nil {
		return i18n.Default
	}
	us.SetLang(stored)
	return i18n.Parse(stored)
}

// t is the short form every handler uses: h.t(chatID, "русский", "english").
func (h *Handler) t(chatID int64, ru, en string) string {
	return i18n.T(h.lang(chatID), ru, en)
}

func (h *Handler) handleLanguage(chatID int64, messageID int) {
	lang := h.lang(chatID)

	// ⚠ Each option is written in its OWN language. A list that reads
	// «Русский / Английский» is unreadable to the person who needs the second
	// one — and that is exactly who is looking at this screen.
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(mark(lang == i18n.RU)+i18n.Name(i18n.RU), "lang_ru"),
			tgbotapi.NewInlineKeyboardButtonData(mark(lang == i18n.EN)+i18n.Name(i18n.EN), "lang_en"),
		},
		{tgbotapi.NewInlineKeyboardButtonData(
			i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "settings_menu")},
	}

	// ⚠ The note is part of the same string rather than a second message: it
	// answers the question this screen provokes — "will my keywords stop
	// working?" — and an answer people have to scroll for is an answer nobody
	// reads.
	body := i18n.T(lang,
		"🌐 <b>Язык</b>\n\nЯзык кнопок и сообщений бота.\n\n"+
			"⚠ Слова, которые бот <b>понимает</b> при создании, работают на обоих языках всегда: от этой настройки они не зависят.",
		"🌐 <b>Language</b>\n\nThe language of the bot's buttons and messages.\n\n"+
			"⚠ The words the bot <b>understands</b> when you create something work in both languages regardless of this setting.")

	h.editOrSend(chatID, messageID, body, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func mark(current bool) string {
	if current {
		return "✅ "
	}
	return ""
}

func (h *Handler) handleLanguageSet(chatID int64, messageID int, lang string) {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetBotLang(us.AuthToken, lang); err != nil {
		// Nothing was written: SetBotLang refuses to patch a blob it could not
		// read (gotcha 21). Saying so matters — the screen would otherwise
		// show the new language selected and lose it on the next read.
		h.editOrSend(chatID, messageID, h.t(chatID,
			"❌ Не удалось сохранить. Настройки не изменились.",
			"❌ Could not save. Nothing was changed."), keyboards.SettingsMenu(h.lang(chatID)))
		return
	}
	us.SetLang(lang)
	h.handleLanguage(chatID, messageID)
}
