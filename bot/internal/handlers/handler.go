package handlers

import (
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/auth"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// handleNotificationAction performs a button press from a reminder message.
//
// Runs on the service token rather than a user session: the API identifies the
// person by the Telegram id that pressed the button and checks that the
// reminder is theirs.
func (h *Handler) handleNotificationAction(chatID int64, from *tgbotapi.User, action notifier.Callback) {
	if from == nil {
		h.sendText(chatID, "⚠️ Не понимаю, от кого это сообщение.")
		return
	}
	if h.cfg.ServiceToken == "" {
		// Nothing to fail loudly about in the chat: the buttons only exist
		// because the notifier sent the message, which needs the same token.
		log.Printf("notification action: SERVICE_TOKEN not set, ignoring %s", action.Action)
		return
	}

	if err := h.api.NotificationAction(
		h.cfg.ServiceToken, from.ID, action.ReminderID, action.Action, action.Minutes,
	); err != nil {
		log.Printf("notification action %s for %s failed: %v", action.Action, action.ReminderID, err)
		h.sendText(chatID, "⚠️ Не получилось — попробуй ещё раз.")
		return
	}

	switch action.Action {
	case notifier.ActionSnooze:
		h.sendText(chatID, "⏰ Напомню через 10 минут.")
	case notifier.ActionDone:
		h.sendText(chatID, "✅ Готово.")
	case notifier.ActionAck:
		h.sendText(chatID, "👌")
	}
}

type Handler struct {
	bot   *tgbotapi.BotAPI
	api   *api.Client
	store *state.Store
	cfg   config.Config
}

func New(bot *tgbotapi.BotAPI, apiClient *api.Client, store *state.Store, cfg config.Config) *Handler {
	return &Handler{bot: bot, api: apiClient, store: store, cfg: cfg}
}

// authRefreshWindow re-logs in slightly before expiry, so a command issued at
// the boundary does not fail on a token that dies mid-request.
const authRefreshWindow = 2 * time.Minute

// ensureAuth gives this chat a JWT before any handler tries to use one.
//
// Until now nothing ever assigned UserState.AuthToken: it was read in seven
// places and written in none, so every /today and every task command went to
// the API with an empty Authorization header and came back unauthorised. The
// bot half of the product was dead while looking healthy.
//
// Returns false when the user cannot be identified or the API rejects the
// login — callers must not proceed, or they will produce the same silent
// unauthorised call this exists to prevent.
func (h *Handler) ensureAuth(chatID int64, from *tgbotapi.User) bool {
	if from == nil {
		// Channel posts and some service messages carry no sender; there is no
		// identity to log in as.
		h.sendText(chatID, "⚠️ Can't identify you from this chat. Message the bot directly.")
		return false
	}

	us := h.store.GetOrCreate(chatID)
	if us.AuthToken != "" && time.Now().Add(authRefreshWindow).Unix() < us.AuthExpiresAt {
		return true
	}

	payload := auth.BuildLoginRequest(h.cfg.TelegramToken, from.ID,
		from.FirstName, from.LastName, from.UserName, "", time.Now().Unix())

	token, expiresAt, err := h.api.TelegramLogin(payload)
	if err != nil {
		log.Printf("auth: login for chat %d failed: %v", chatID, err)
		h.sendText(chatID, "⚠️ Couldn't sign you in. Try again in a minute.")
		return false
	}

	h.store.SetAuth(chatID, token, expiresAt)
	return true
}

func (h *Handler) HandleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	if !h.ensureAuth(chatID, msg.From) {
		return
	}
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
	case "📅 New Event":
		h.startNewEventFlow(chatID)
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
	// 🔴 CallbackQuery.Message is OPTIONAL in the Bot API. Telegram omits it
	// when the message is too old, and for callbacks from inline messages. This
	// line dereferenced it unconditionally, so anyone pressing a button under
	// an old reminder took the whole process down — for every user, on a host
	// where the bot is redeployed by hand.
	//
	// Note the asymmetry that made it easy to miss: cb.From IS nil-checked, two
	// call sites below.
	//
	// The callback still has to be answered. Without that, Telegram leaves the
	// spinner turning on the user's button forever, which reads as a hung bot
	// rather than as a stale message.
	if cb.Message == nil {
		if _, err := h.bot.Request(tgbotapi.NewCallback(cb.ID, "This message is too old — open the bot and try again")); err != nil {
			log.Printf("callback %s: could not answer a message-less callback: %s", cb.ID, logsafe.Redact(err))
		}
		return
	}

	chatID := cb.Message.Chat.ID
	data := cb.Data

	// Answering the callback is what stops the spinner on the user's button;
	// failing silently here looks exactly like a hung bot.
	if _, err := h.bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		log.Printf("callback %s: answer failed: %s", cb.ID, logsafe.Redact(err))
	}

	// Notification buttons are handled BEFORE ensureAuth: they travel on the
	// service token, not on a user JWT, so a failure to mint a user session must
	// not stop someone snoozing a reminder.
	if action, ok := notifier.ParseCallback(data); ok {
		h.handleNotificationAction(chatID, cb.From, action)
		return
	}

	if !h.ensureAuth(chatID, cb.From) {
		return
	}

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
	case strings.HasPrefix(data, "when_"):
		h.handleWhenSelect(chatID, data)
	}
}

// send is the one place a message actually leaves the bot, so it is the one
// place the error can be noticed.
//
// Every caller used to drop it. A user who blocked the bot, a message over
// Telegram's length limit, malformed HTML — all produced silence in the chat
// AND silence in the log, which makes "the bot did not answer" impossible to
// diagnose after the fact. The notifier already got this right
// (notifier.go:73); the interactive half did not.
//
// Redacted: a transport failure arrives as *url.Error carrying the API URL,
// token included.
func (h *Handler) send(chatID int64, msg tgbotapi.Chattable) {
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("send to chat %d failed: %s", chatID, logsafe.Redact(err))
	}
}

func (h *Handler) sendText(chatID int64, text string) {
	h.send(chatID, tgbotapi.NewMessage(chatID, text))
}

func (h *Handler) sendHTML(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.send(chatID, msg)
}

func (h *Handler) sendHTMLWithKeyboard(chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = kb
	h.send(chatID, msg)
}
