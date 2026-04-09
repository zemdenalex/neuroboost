package handlers

import (
	"strings"
	"time"

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
	chatID := msg.Chat.ID
	us := h.store.GetOrCreate(chatID)

	if us.CurrentFlow != "" {
		h.handleFlowInput(chatID, msg.Text)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start", "help":
			h.handleStart(chatID)
		default:
			h.sendText(chatID, "Unknown command. Use /start to see the menu.")
		}
		return
	}

	switch msg.Text {
	case "🎯 Today":
		h.handleToday(chatID)
	case "📋 Tasks":
		h.handleTasks(chatID)
	case "📝 Note":
		h.startNoteFlow(chatID)
	case "➕ New Task":
		h.startNewTaskFlow(chatID)
	case "🗓 Calendar":
		h.handleCalendar(chatID, time.Now())
	case "📊 Stats":
		h.handleStats(chatID)
	case "⚙️ Settings":
		h.handleSettings(chatID)
	default:
		h.sendText(chatID, "Use the menu buttons or /start")
	}
}

func (h *Handler) HandleCallback(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	data := cb.Data

	h.bot.Send(tgbotapi.NewCallback(cb.ID, ""))

	switch {
	case data == "main_menu":
		h.handleStart(chatID)
	case data == "today_focus":
		h.handleToday(chatID)
	case data == "top_tasks":
		h.handleTasks(chatID)
	case strings.HasPrefix(data, "task_action_"):
		h.handleTaskAction(chatID, strings.TrimPrefix(data, "task_action_"))
	case strings.HasPrefix(data, "task_done_"):
		h.handleTaskDone(chatID, strings.TrimPrefix(data, "task_done_"))
	case strings.HasPrefix(data, "task_delete_"):
		h.handleTaskDelete(chatID, strings.TrimPrefix(data, "task_delete_"))
	case strings.HasPrefix(data, "priority_"):
		h.handlePrioritySelect(chatID, data)
	}
}

func (h *Handler) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	h.bot.Send(msg)
}

func (h *Handler) sendHTML(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *Handler) sendHTMLWithKeyboard(chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = kb
	h.bot.Send(msg)
}
