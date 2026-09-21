package handlers

import (
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/auth"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

// handleNotificationAction performs a button press from a reminder message.
//
// Runs on the service token rather than a user session: the API identifies the
// person by the Telegram id that pressed the button and checks that the
// reminder is theirs.
func (h *Handler) handleNotificationAction(chatID int64, from *tgbotapi.User, msg *tgbotapi.Message, action notifier.Callback) {
	if from == nil {
		h.sendText(chatID, h.t(chatID, "⚠️ Не понимаю, от кого это сообщение.", "⚠️ I can't tell who sent this."))
		return
	}
	// «⏰ Своё» is a question, not a postponement: it never reaches the API.
	// Denis, 18.09, looking at two fixed buttons: «и где свой вариант?»
	if action.Action == notifier.ActionSnoozeAsk {
		h.askSnoozeInterval(chatID, action.ReminderID)
		return
	}

	if h.cfg.ServiceToken == "" {
		// Nothing to fail loudly about in the chat: the buttons only exist
		// because the notifier sent the message, which needs the same token.
		log.Printf("notification action: SERVICE_TOKEN not set, ignoring %s", action.Action)
		return
	}

	outcome, err := h.api.NotificationAction(
		h.cfg.ServiceToken, from.ID, action.ReminderID, action.Action, action.Minutes,
	)
	if err != nil {
		log.Printf("notification action %s for %s failed: %v", action.Action, action.ReminderID, err)
		h.sendText(chatID, h.t(chatID, "⚠️ Не получилось — попробуй ещё раз.", "⚠️ That didn't work — try again."))
		return
	}

	// 🔴 One line, from a pure function, instead of a switch that has to be
	// remembered. The switch that used to live here covered snooze, done and
	// ack — and when INVITE buttons arrived on 17.08 it covered neither of
	// them. The API answered 200 to every press and the chat said nothing, so
	// the feature looked broken and got pressed seven times. The set of replies
	// is now held against the set of buttons by TestEveryButtonHasAReply.
	if reply := notifier.ActionReply(h.lang(chatID), action.Action, action.Minutes); reply != "" {
		h.sendText(chatID, reply)
	}

	// 🔴 A press can lead to another question. Answering «какой аккаунт
	// оставить» when both sides own a personal calendar is only half the
	// decision, and stripping the buttons here would end the conversation in
	// the middle of it.
	if outcome.Ask == "calendar" {
		h.askAboutPersonalCalendars(chatID, msg, action.ReminderID)
		return
	}
	if outcome.Merged {
		h.sendText(chatID, h.t(chatID,
			"🔗 Аккаунты объединены. Теперь вход и через email, и через Telegram.",
			"🔗 Accounts merged. You can now sign in with either email or Telegram."))
	}

	// Take the buttons off the message that was just answered. Leaving them
	// invites a second press, and a second press cannot undo the first — an
	// accepted invitation stays accepted, so "Отклонить" underneath it is a
	// control that no longer does what it says.
	if msg != nil {
		empty := keyboards.None()
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

	// quickKind is the kind a qa_ button chose, consumed by the next parse.
	// One update is handled at a time, so it needs no lock.
	quickKind string
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
		h.sendText(chatID, h.t(chatID, "⚠️ Не понимаю, кто ты в этом чате. Напиши боту напрямую.", "⚠️ I can't identify you in this chat. Message the bot directly."))
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
		h.sendText(chatID, h.t(chatID, "⚠️ Не получилось войти. Попробуй через минуту.", "⚠️ Could not sign you in. Try in a minute."))
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

	// 🔴 A menu button is a button, even in the middle of creating something.
	//
	// This check used to come AFTER the flow check, and that order was the
	// whole defect Denis reported on 15.09: «🗓 Календарь» pressed while an
	// event was being created arrived as the event's TITLE. A reply button is
	// an ordinary text message as far as the Bot API is concerned, so nothing
	// distinguishes it except asking — and asking late is the same as not
	// asking.
	//
	// Abandoning the flow is deliberate rather than a pause: the user reached
	// for a different part of the product, and a half-built draft waiting
	// silently to swallow the next message is worse than losing it.
	if screen, ok := keyboards.MenuScreen(msg.Text); ok {
		// 🔴 A menu press in the middle of onboarding is «skip», not a trap —
		// the same rule as for every other flow, plus the flag, or the next
		// press would start it all over again.
		if us.CurrentFlow == onboardFlow {
			h.finishOnboarding(chatID)
		}
		if us.CurrentFlow != "" {
			h.store.ClearFlow(chatID)
		}
		if h.needsOnboarding(chatID) {
			h.startOnboarding(chatID, 0, msg.From.LanguageCode)
			return
		}
		h.openScreen(chatID, screen)
		return
	}

	if us.CurrentFlow != "" {
		h.handleFlowInput(chatID, msg.Text)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start", "help":
			// 🔴 An invite link before onboarding, always. The invitation is
			// why this person opened the bot at all; a language question in
			// front of it loses the token — /start carries it exactly once —
			// and reads as the bot ignoring the link they were sent.
			if payload := strings.TrimSpace(msg.CommandArguments()); strings.HasPrefix(payload, "inv_") {
				h.askInviteLink(chatID, strings.TrimPrefix(payload, "inv_"))
				return
			}
			if h.needsOnboarding(chatID) {
				h.startOnboarding(chatID, 0, msg.From.LanguageCode)
				return
			}
			h.handleStart(chatID)
		default:
			h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Неизвестная команда.", "Unknown command."), keyboards.HomeInline(h.lang(chatID)))
		}
		return
	}

	// Text from nowhere is a request to create something — quickadd.go.
	if strings.TrimSpace(msg.Text) != "" {
		h.handleQuickAdd(chatID, msg.Text)
		return
	}

	h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял. Вот меню:", "Didn't get that. Here's the menu:"), keyboards.HomeInline(h.lang(chatID)))
}

// openScreen renders the screen a reply-keyboard button names.
//
// It exists as its own function so the dispatch is one list rather than a
// switch buried inside HandleMessage's other branches — TestEveryMenuScreenHasACase
// reads it, and a button added to the keyboard without a case here fails loudly
// instead of clearing the flow and then rendering nothing.
func (h *Handler) openScreen(chatID int64, screen string) {
	switch screen {
	case keyboards.ScreenMenu:
		h.handleMenu(chatID, 0)
	case keyboards.ScreenCalendar:
		h.handleCalendar(chatID, 0, time.Now())
	case keyboards.ScreenAgenda:
		h.handleAgenda(chatID, 0)
	case keyboards.ScreenTasks:
		h.handleTasks(chatID, 0)
	case keyboards.ScreenCreate:
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Что создаём?", "What are we creating?"), keyboards.CreateMenu(h.lang(chatID)))
	case keyboards.ScreenSettings:
		h.handleSettings(chatID, 0)
	default:
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял. Вот меню:", "Didn't get that. Here's the menu:"), keyboards.HomeInline(h.lang(chatID)))
	}
}

// bilingualTooOld is the one user-facing string in this bot that carries both
// languages at once.
//
// 🔴 It is answered BEFORE ensureAuth, on a callback whose message Telegram no
// longer sends us. There is no session, so there is no language to read — and
// reading one would mean an API call on the path whose whole purpose is to stop
// the spinner immediately. Both languages in one line is the honest answer;
// picking one silently would be a guess.
const bilingualTooOld = "Сообщение слишком старое / This message is too old"

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
		if _, err := h.bot.Request(tgbotapi.NewCallback(cb.ID, bilingualTooOld)); err != nil {
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

	// The confirmation card owns every dr_/dre_ callback. It is asked before
	// the switch rather than inside it because the card has a dozen buttons
	// with three prefixes, and a dozen more cases in a switch this long is how
	// one of them ends up unreachable.
	// Onboarding owns ob_.
	if h.handleOnboardCallback(chatID, cb.Message.MessageID, data, cb.From) {
		return
	}

	// Quick add's question owns qa_; it hands over to the card's flows.
	if h.handleQuickCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	if h.handleDraftCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	// The keyword screen owns kwf_/kwv_. Its own prefixes, not the card's —
	// see keyboards/trigger.go for why reusing dr_col_ would have been a bug.
	if h.handleKeywordCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	// The event screens own ev_/eve_/evd_/evdy_ and the picker. Asked before
	// the switch for the same reason the card is: four prefixes that differ by
	// one letter belong next to each other, not scattered through a long switch
	// where one of them ends up unreachable.
	if h.handleEventCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	// The calendar screens own cals/cal_. Asked here rather than in the switch
	// for the same reason as the others: the prefixes overlap each other
	// («cal_» is a prefix of «cal_new»), and ordering that matters belongs in
	// one place where it can be read at a glance.
	if h.handleCalendarsCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	// Feedback and release notes own fb/fb_* and whatsnew*.
	if h.handleLinkingCallback(chatID, cb.Message.MessageID, data) {
		return
	}
	if h.handleFeedbackCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	// «Своё» on a reminder owns snz_*.
	if h.handleSnoozeCallback(chatID, cb.Message.MessageID, data) {
		return
	}

	switch {
	case data == "main_menu":
		// 🔴 Going home ENDS whatever was being written. Denis, 18.09: «это
		// работает даже если на этапе создания нажать отмена, он вернется в
		// меню, но следующая задача все равно не создастся» — the menu was
		// drawn while the flow kept running, so the next line was read as an
		// answer to a question no longer on screen, and answered «Что-то пошло
		// не так».
		//
		// «❌ Отмена» on the task card sends exactly this callback, which is
		// why cancelling has to mean cancelling here rather than in each card.
		h.store.ClearFlow(chatID)
		h.handleMenu(chatID, cb.Message.MessageID)
	case data == "stats":
		h.handleStatsView(chatID, cb.Message.MessageID, statsView{Period: statgrid.Week, Entity: "a"})
	// Statistics (22.09): the screen's state rides in the button.
	case strings.HasPrefix(data, "stsc_"):
		if v, ok := parseStatsView(strings.TrimPrefix(data, "stsc_")); ok {
			h.handleStatsScale(chatID, cb.Message.MessageID, v)
		}
	case strings.HasPrefix(data, "st_"):
		if v, ok := parseStatsView(strings.TrimPrefix(data, "st_")); ok {
			h.handleStatsView(chatID, cb.Message.MessageID, v)
		}
	case data == "create_menu":
		h.editOrSend(chatID, cb.Message.MessageID, h.t(chatID, "Что создаём?", "What are we creating?"), keyboards.CreateMenu(h.lang(chatID)))
	case data == "new_task":
		h.startNewTaskFlow(chatID)
	case data == "new_event":
		h.startNewEventFlow(chatID)
	case data == "new_note":
		h.startNoteFlow(chatID)
	case data == "today_focus":
		h.handleToday(chatID, cb.Message.MessageID)
	case data == "top_tasks":
		h.handleTasks(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "task_action_"):
		h.handleTaskAction(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_action_"))
	// The three scheduling steps, in the order they fire. They sit above
	// task_done_/task_delete_ only for readability — every prefix here is
	// distinct, deliberately: a switch on prefixes where one is a prefix of
	// another routes by declaration order, which is a rule nobody remembers
	// when adding the fourth button.
	// Task ↔ event, the full path (21.09). t2e_ opens it; every later step is
	// a short t2 code, the choice itself living in the flow.
	// Event → task (21.09), the same shape with an e2 prefix.
	case strings.HasPrefix(data, "e2t_"):
		h.handleToTaskStart(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "e2t_"))
	case strings.HasPrefix(data, "e2"):
		h.handleToTaskStep(chatID, cb.Message.MessageID, data)
	case strings.HasPrefix(data, "t2e_"):
		h.handleToEventStart(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "t2e_"))
	case strings.HasPrefix(data, "t2"):
		h.handleToEventStep(chatID, cb.Message.MessageID, data)
	case strings.HasPrefix(data, "task_sched_"):
		h.handleTaskScheduleWhen(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_sched_"))
	case strings.HasPrefix(data, "task_when_"):
		h.handleTaskScheduleDuration(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_when_"))
	case strings.HasPrefix(data, "task_plan_"):
		h.handleTaskSchedule(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_plan_"))
	case strings.HasPrefix(data, "task_ppd_"):
		// task_ppd_<id>_<days> — the id is a UUID and carries no underscore, so
		// the last one separates the two.
		rest := strings.TrimPrefix(data, "task_ppd_")
		if i := strings.LastIndex(rest, "_"); i > 0 {
			if days, err := strconv.Atoi(rest[i+1:]); err == nil {
				h.handleTaskPostponeDays(chatID, cb.Message.MessageID, rest[:i], days)
			}
		}
	case strings.HasPrefix(data, "task_ppc_"):
		h.handleTaskPostponeCustom(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_ppc_"))
	case strings.HasPrefix(data, "task_pp_"):
		h.handleTaskPostpone(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_pp_"))
	// 🔁 Повтор / 🔔 Долбить on an existing task. Same shape as task_ppd_ /
	// task_pp_ above, and the same ordering rule: the value-carrying prefix is
	// declared BEFORE the screen's, or «task_rpd_…» would be eaten by a
	// «task_rp» test. It is not eaten today — the next character is «d», not
	// «_» — but that is an accident of naming, and this switch is read as an
	// ordered list, so the order is where the guarantee belongs.
	case strings.HasPrefix(data, "task_rpd_"):
		rest := strings.TrimPrefix(data, "task_rpd_")
		if i := strings.LastIndex(rest, "_"); i > 0 {
			h.handleTaskRepeatSet(chatID, cb.Message.MessageID, rest[:i], rest[i+1:])
		}
	case strings.HasPrefix(data, "task_rp_"):
		h.handleTaskRepeatMenu(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_rp_"))
	case strings.HasPrefix(data, "task_ngd_"):
		rest := strings.TrimPrefix(data, "task_ngd_")
		if i := strings.LastIndex(rest, "_"); i > 0 {
			h.handleTaskNagSet(chatID, cb.Message.MessageID, rest[:i], rest[i+1:])
		}
	case strings.HasPrefix(data, "task_ng_"):
		h.handleTaskNagMenu(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_ng_"))
	case strings.HasPrefix(data, "task_done_"):
		h.handleTaskDone(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_done_"))
	case strings.HasPrefix(data, "task_delete_"):
		h.handleTaskDelete(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_delete_"))
	// Срок / Оценка / Теги. The "_set_" variants are the wizard-style screens'
	// own answer buttons and must be checked before their plain "task_due_" /
	// "task_est_" prefix — same declaration-order rule as above.
	case strings.HasPrefix(data, "task_due_set_"):
		h.handleTaskDueSet(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_due_set_"))
	case strings.HasPrefix(data, "task_due_"):
		h.handleTaskDueMenu(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_due_"))
	case strings.HasPrefix(data, "task_est_set_"):
		h.handleTaskEstimateSet(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_est_set_"))
	case strings.HasPrefix(data, "task_est_"):
		h.handleTaskEstimateMenu(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_est_"))
	case strings.HasPrefix(data, "task_tag_"):
		h.handleTaskTagsPrompt(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "task_tag_"))
	// The month grid. Every step of it — paging, opening a day, and going back
	// to the month — edits the message it was pressed on. cal_back_ used to post
	// a new one, and the comment that stood here explained why: the day view was
	// "its own message". The day view edits in place too now, so grid and day
	// share one message and the exception no longer had anything to protect.
	case data == "cal_today":
		h.handleCalendarDay(chatID, cb.Message.MessageID, time.Now().In(h.location(chatID)).Format("2006-01-02"))
	case strings.HasPrefix(data, "cal_new_"):
		h.startNewEventForDay(chatID, strings.TrimPrefix(data, "cal_new_"))
	case strings.HasPrefix(data, "cal_prev_"), strings.HasPrefix(data, "cal_next_"):
		h.handleCalendarNav(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "cal_"))
	case strings.HasPrefix(data, "cal_day_"):
		h.handleCalendarDay(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "cal_day_"))
	case strings.HasPrefix(data, "cal_back_"):
		h.handleCalendarBack(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "cal_back_"))
	// The grid's header and weekday cells. Telegram has no inert button, so they
	// carry "noop" — and it must land somewhere, or every tap on a weekday
	// header falls through to the default and does nothing visible while the
	// callback has already been answered above.
	case data == "noop":
		return
	case data == "cal_open":
		h.handleCalendar(chatID, cb.Message.MessageID, time.Now())
	case data == "agenda_open":
		h.handleAgenda(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "guide_full_"):
		h.handleGuideFull(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "guide_full_"))
	case data == "planning":
		h.handlePlanning(chatID, cb.Message.MessageID)
	case data == "settings_menu":
		h.handleSettings(chatID, cb.Message.MessageID)
	case data == "settings_lang":
		h.handleLanguage(chatID, cb.Message.MessageID)
	case data == "lang_ru":
		h.handleLanguageSet(chatID, cb.Message.MessageID, string(i18n.RU))
	case data == "lang_en":
		h.handleLanguageSet(chatID, cb.Message.MessageID, string(i18n.EN))
	case data == "settings_keywords":
		h.handleKeywords(chatID, cb.Message.MessageID)
	case data == "kw_add":
		h.startKeywordFlow(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "kw_del_"):
		h.handleKeywordDelete(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "kw_del_"))
	case data == "settings_workhours":
		h.handleWorkHours(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "wh_"):
		h.handleWorkHourSet(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "wh_"))
	case data == "nt_save":
		h.handleTaskCardSave(chatID, cb.Message.MessageID)
	case data == "nt_wizard":
		h.handleTaskWizardStart(chatID, cb.Message.MessageID)
	case data == "nt_skip":
		h.handleTaskWizardSkip(chatID, cb.Message.MessageID)
	case strings.HasPrefix(data, "nt_p_"):
		h.handleWizardPriority(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "nt_p_"))
	case strings.HasPrefix(data, "nt_d_"):
		h.handleWizardDue(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "nt_d_"))
	case strings.HasPrefix(data, "nt_r_"):
		h.handleWizardRepeat(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "nt_r_"))
	case strings.HasPrefix(data, "nt_e_"):
		h.handleWizardEstimate(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "nt_e_"))
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
	// No buttons means no markup at all on a new message — and never a nil
	// keyboard, which Telegram refuses outright (keyboards.None).
	if len(kb.InlineKeyboard) > 0 {
		msg.ReplyMarkup = kb
	}
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
	if kb.InlineKeyboard == nil {
		kb = keyboards.None()
	}
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
