package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleStart(chatID int64) {
	text := "🧠 <b>NeuroBoost</b>\n\n" +
		"Calendar-first productivity for neurodivergent minds.\n\n" +
		"Use the menu below to get started:"
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboards.MainMenu()
	h.bot.Send(msg)
}

func (h *Handler) handleSettings(chatID int64) {
	text := "⚙️ <b>Settings</b>\n\n" +
		"Settings are managed in the web app.\n" +
		"Open: https://neuroboost.website/settings"
	h.sendHTML(chatID, text)
}
