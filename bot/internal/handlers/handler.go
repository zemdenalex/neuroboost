package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

type Handler struct {
	bot   *tgbotapi.BotAPI
	api   *api.Client
	store *state.Store
	cfg   config.Config
}

func New(bot *tgbotapi.BotAPI, apiClient *api.Client, store *state.Store, cfg config.Config) *Handler {
	return &Handler{bot: bot, api: apiClient, store: store, cfg: cfg}
}

func (h *Handler) HandleMessage(msg *tgbotapi.Message) {
	// Stub — will be implemented in Task 4
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Bot is starting up. Use /start soon!")
	h.bot.Send(reply)
}

func (h *Handler) HandleCallback(cb *tgbotapi.CallbackQuery) {
	// Stub — will be implemented in Task 4
	h.bot.Send(tgbotapi.NewCallback(cb.ID, ""))
}
