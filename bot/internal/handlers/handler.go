package handlers

import (
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/auth"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// handleNotificationAction performs a button press from a reminder message.
//
// Runs on the service token rather than a user session: the API identifies the
// person by the Telegram id that pressed the button and checks that the
// reminder is theirs.
func (h *Handler) handleNotificationAction(chatID int64, from *tgbotapi.User, msg *tgbotapi.Message, action notifier.Callback) {
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

	// 🔴 One line, from a pure function, instead of a switch that has to be
	// remembered. The switch that used to live here covered snooze, done and
	// ack — and when INVITE buttons arrived on 17.08 it covered neither of
	// them. The API answered 200 to every press and the chat said nothing, so
	// the feature looked broken and got pressed seven times. The set of replies
	// is now held against the set of buttons by TestEveryButtonHasAReply.
	if reply := notifier.ActionReply(action.Action); reply != "" {
		h.sendText(chatID, reply)
	}

	// Take the buttons off the message that was just answered. Leaving them
	// invites a second press, and a second press cannot undo the first — an
	// accepted invitation stays accepted, so "Отклонить" underneath it is a
	// control that no longer does what it says.
	if msg != nil {
		empty := tgbotapi.NewInlineKeyboardMarkup()
		edit := tgbotapi.NewEditMessageReplyMarkup(chatID, msg.MessageID, empty)
		if _, err := h.bot.Request(edit); err != nil {
			// Not worth telling the user: the action itself succeeded and the
			// reply above already said so. Telegram refuses an edit for its own
			// reasons (message too old, identical markup) and none of them mean
			// the press failed.
			log.Printf("clearing buttons on message %d failed: %s", msg.MessageID, logsafe.Redact(err))
		}
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
			h.sendHTMLWithKeyboard(chatID, "Неизвестная команда.", keyboards.HomeInline())
		}
		return
	}

	switch msg.Text {
	case "🏠 Меню":
		h.handleMenu(chatID, 0)
	case "🗓 Календарь":
		h.handleCalendar(chatID, time.Now())
	case "📅 События":
		h.handleAgenda(chatID, 0)
	case "📋 Задачи":
		h.handleTasks(chatID, 0)
	case "➕ Создать":
		h.sendHTMLWithKeyboard(chatID, "Что создаём?", keyboards.CreateMenu())
	case "⚙️ Настройки":
		h.handleSettings(chatID)
	default:
		h.sendHTMLWithKeyboard(chatID, "Не понял. Вот меню:", keyboards.HomeInline())
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
		h.handleNotificationAction(chatID, cb.From, cb.Message, action)
		return
	}

	if !h.ensureAuth(chatID, cb.From) {
		return
	}

	switch {
	case data == "main_menu":
		h.handleMenu(chatID, cb.Message.MessageID)
	case data == "stats":
		h.handleStats(chatID)
	case data == "create_menu":
		h.editOrSend(chatID, cb.Message.MessageID, "Что создаём?", keyboards.CreateMenu())
	case data == "new_task":
		h.startNewTaskFlow(chatID)
	case data == "new_event":
		h.startNewEventFlow(chatID)
	case data == "new_note":
		h.startNoteFlow(chatID)
	case data == "today_focus":
		h.handleToday(chatID)
	case data == "top_tasks":
		h.handleTasks(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "task_action_"):
		h.handleTaskAction(chatID, strings.TrimPrefix(data, "task_action_"))
	// The three scheduling steps, in the order they fire. They sit above
	// task_done_/task_delete_ only for readability — every prefix here is
	// distinct, deliberately: a switch on prefixes where one is a prefix of
	// another routes by declaration order, which is a rule nobody remembers
	// when adding the fourth button.
	case strings.HasPrefix(data, "task_sched_"):
		h.handleTaskScheduleWhen(chatID, strings.TrimPrefix(data, "task_sched_"))
	case strings.HasPrefix(data, "task_when_"):
		h.handleTaskScheduleDuration(chatID, strings.TrimPrefix(data, "task_when_"))
	case strings.HasPrefix(data, "task_plan_"):
		h.handleTaskSchedule(chatID, strings.TrimPrefix(data, "task_plan_"))
	case strings.HasPrefix(data, "task_done_"):
		h.handleTaskDone(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_done_"))
	case strings.HasPrefix(data, "task_delete_"):
		h.handleTaskDelete(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_delete_"))
	// The month grid. cal_back_ re-renders as a NEW message rather than editing:
	// it is pressed from a day view, which is its own message, and editing that
	// into a grid would destroy what the user was reading.
	case strings.HasPrefix(data, "cal_prev_"), strings.HasPrefix(data, "cal_next_"):
		h.handleCalendarNav(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "cal_"))
	case strings.HasPrefix(data, "cal_day_"):
		h.handleCalendarDay(chatID, strings.TrimPrefix(data, "cal_day_"))
	case strings.HasPrefix(data, "cal_back_"):
		h.handleCalendarBack(chatID, strings.TrimPrefix(data, "cal_back_"))
	// The grid's header and weekday cells. Telegram has no inert button, so they
	// carry "noop" — and it must land somewhere, or every tap on a weekday
	// header falls through to the default and does nothing visible while the
	// callback has already been answered above.
	case data == "noop":
		return
	case data == "cal_open":
		h.handleCalendar(chatID, time.Now())
	case data == "planning":
		h.handlePlanning(chatID)
	case data == "settings_menu":
		h.handleSettings(chatID)
	case data == "settings_workhours":
		h.handleWorkHours(chatID)
	case strings.HasPrefix(data, "wh_"):
		h.handleWorkHourSet(chatID, strings.TrimPrefix(data, "wh_"))
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

// editHTMLWithKeyboard replaces a message in place — what paging needs, so a
// month calendar does not leave a new grid in the chat on every tap.
//
// Telegram answers "message is not modified" when the new text and keyboard are
// byte-identical to the old, which happens whenever a user taps the same page
// twice. That is not a failure worth a log line, and it is certainly not worth
// a message to the user; everything else is.
func (h *Handler) editHTMLWithKeyboard(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
	msg.ParseMode = "HTML"
	if _, err := h.bot.Send(msg); err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return
		}
		log.Printf("edit in chat %d failed: %s", chatID, logsafe.Redact(err))
	}
}

func (h *Handler) sendHTMLWithKeyboard(chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = kb
	h.send(chatID, msg)
}

// shouldSendNew decides whether a navigation step must post a new message.
//
// Pulled out of editOrSend as a pure function on purpose: the method around it
// cannot run without a bot, a store and a reachable API, and this decision is
// the only part of it that can be wrong.
func shouldSendNew(messageID int, editErr error) bool {
	if messageID == 0 {
		// The press came from a reply-keyboard button or a command; there is no
		// message of ours to edit.
		return true
	}
	if editErr == nil {
		return false
	}
	// An identical re-render. Not a failure, and answering it with a fresh
	// message would double the screen on every second tap.
	if strings.Contains(editErr.Error(), "message is not modified") {
		return false
	}
	// Anything else — most often a message past Telegram's 48-hour edit window.
	// Falling back to a new message is the only outcome the user can see.
	return true
}

// editOrSend renders a screen onto the message it was triggered from, or posts
// a new one when there is nothing to edit.
func (h *Handler) editOrSend(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	if messageID != 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
		edit.ParseMode = "HTML"
		_, err := h.bot.Send(edit)
		if !shouldSendNew(messageID, err) {
			return
		}
		if err != nil {
			log.Printf("edit in chat %d fell back to a new message: %s", chatID, logsafe.Redact(err))
		}
	}
	h.sendHTMLWithKeyboard(chatID, text, kb)
}
