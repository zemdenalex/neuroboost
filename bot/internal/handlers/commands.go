package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🧠 <b>NeuroBoost</b>")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboards.MainMenu()
	h.send(chatID, msg)
	h.handleMenu(chatID, 0)
}

// handleSettings moved to settings.go when it stopped being a link.
