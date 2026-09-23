package notifier

import (
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"

	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CallbackPrefix marks a callback that belongs to a notification button.
//
// Short on purpose: Telegram caps callback_data at 64 BYTES, and a reminder id
// is a 36-character UUID. "nb:s:" + 36 leaves comfortable room; spelling the
// action out would not.
const CallbackPrefix = "nb:"

// Action codes as they travel inside callback_data.
const (
	codeAck    = "a"
	codeSnooze = "s"
	codeDone   = "d"
	// Two snooze lengths, two codes.
	//
	// 🔴 The minutes live in the CODE, not in the payload. callback_data is
	// capped at 64 bytes and a reminder id already spends 36 of them; a payload
	// that grows with every new snooze option is a payload that will one day be
	// truncated — and a truncated button makes Telegram refuse the whole
	// message, so the notification never arrives.
	//
	// codeSnooze ("s") stays as the ten-minute meaning: notifications already
	// delivered carry it under buttons on people's phones, and a button that
	// silently stops answering is indistinguishable from a dead bot.
	codeSnoozeHour = "h"
	// «Своё» does not postpone by itself — it opens the question. The bot
	// answers it in the chat, because a free interval cannot be a button.
	codeSnoozeAsk = "k"
	// Answering a calendar invitation. One letter each, for the same reason as
	// the rest: 64 bytes total and 36 of them are a UUID.
	codeAccept  = "y"
	codeDecline = "n"

	// Answering «объединить аккаунты?» (v0.4.11.5). Four one-byte codes, for
	// the same reason as every other code here: callback_data is capped at 64
	// bytes and a UUID already spends 36 of them.
	codeKeepSite = "1"
	codeKeepTg   = "2"
	codeCalMerge = "m"
	codeCalBoth  = "b"
)

// Action names as the API expects them. Kept separate from the wire codes so
// shortening the wire format can never silently rename an API action.
const (
	ActionAck    = "ack"
	ActionSnooze = "snooze"
	// ActionSnoozeAsk never reaches the API: it is handled entirely in the
	// bot, which then sends a normal snooze with the minutes it was told.
	ActionSnoozeAsk = "snooze_ask"
	ActionDone      = "done"
	ActionAccept    = "accept"
	ActionDecline   = "decline"
	// The merge answers must match internal/reminders/action.go exactly: the
	// string travels over HTTP and a typo here is a 400 the person reads as
	// "the button is broken".
	ActionKeepSite = "keep_site"
	ActionKeepTg   = "keep_tg"
	ActionCalMerge = "cal_merge"
	ActionCalBoth  = "cal_both"
)

// SnoozeMinutes is what the short "later" button asks for, and SnoozeHour the
// long one. Both are within the API's cap of a day
// (reminders/action.go:32).
const (
	SnoozeMinutes = 10
	SnoozeHour    = 60
)

// Callback is a decoded notification button press.
type Callback struct {
	Action     string
	ReminderID string
	Minutes    int
}

// EncodeCallback builds the callback_data for one button.
func EncodeCallback(code, reminderID string) string {
	return CallbackPrefix + code + ":" + reminderID
}

// ParseCallback decodes a notification button press.
//
// Returns ok=false for anything that is not ours, so the bot's existing
// callback routing is untouched by data it has always handled.
func ParseCallback(data string) (Callback, bool) {
	if !strings.HasPrefix(data, CallbackPrefix) {
		return Callback{}, false
	}
	rest := strings.TrimPrefix(data, CallbackPrefix)
	code, id, found := strings.Cut(rest, ":")
	if !found || id == "" {
		return Callback{}, false
	}
	switch code {
	case codeAck:
		return Callback{Action: ActionAck, ReminderID: id}, true
	case codeSnooze:
		return Callback{Action: ActionSnooze, ReminderID: id, Minutes: SnoozeMinutes}, true
	case codeSnoozeHour:
		return Callback{Action: ActionSnooze, ReminderID: id, Minutes: SnoozeHour}, true
	case codeSnoozeAsk:
		return Callback{Action: ActionSnoozeAsk, ReminderID: id}, true
	case codeDone:
		return Callback{Action: ActionDone, ReminderID: id}, true
	case codeAccept:
		return Callback{Action: ActionAccept, ReminderID: id}, true
	case codeDecline:
		return Callback{Action: ActionDecline, ReminderID: id}, true
	case codeKeepSite:
		return Callback{Action: ActionKeepSite, ReminderID: id}, true
	case codeKeepTg:
		return Callback{Action: ActionKeepTg, ReminderID: id}, true
	case codeCalMerge:
		return Callback{Action: ActionCalMerge, ReminderID: id}, true
	case codeCalBoth:
		return Callback{Action: ActionCalBoth, ReminderID: id}, true
	}
	return Callback{}, false
}

// Keyboard builds the buttons for one notification.
//
// A digest gets none: it is a summary of several things, so "done" and "later"
// have no single subject to act on.
// Keyboard builds the buttons for one notification, in the recipient's language.
//
// ⚠ This comment used to say the opposite: that the buttons were Russian-only
// and structurally had to be. The reasoning was sound — the NOTIFIER runs on the
// service token and knows the recipient only as a Telegram id, with no session,
// no settings and deliberately no access to the store, so reading a language
// here would mean an API call per notification on a path that runs every minute
// for every user.
//
// It named its own fix: «the honest fix is for the API to send the language
// alongside the pending notification». That is what happens now — the API
// already has the row and answers with it (reminders/service.go), and `lang`
// arrives in the payload. The gap it described is closed; the reasoning is kept
// because it is why the answer is shaped this way.
func Keyboard(sourceKind, reminderID, lang string) *tgbotapi.InlineKeyboardMarkup {
	l := i18n.RU
	if lang == "en" {
		l = i18n.EN
	}
	var row []tgbotapi.InlineKeyboardButton

	switch strings.ToUpper(sourceKind) {
	case "TASK":
		row = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "✅ Готово", "✅ Done"), EncodeCallback(codeDone, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ 10 мин", "⏰ 10 min"), EncodeCallback(codeSnooze, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ Час", "⏰ An hour"), EncodeCallback(codeSnoozeHour, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ Своё", "⏰ Custom"), EncodeCallback(codeSnoozeAsk, reminderID)),
		}
	case "INVITE":
		// 🔴 No snooze. Snoozing re-sends the same notification later, and an
		// invitation is not a reminder — the answer is yes or no, and a third
		// button offering neither would make the message look like a chore to
		// postpone rather than a question to answer.
		row = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "✅ Принять", "✅ Accept"), EncodeCallback(codeAccept, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "✖ Отклонить", "✖ Decline"), EncodeCallback(codeDecline, reminderID)),
		}

	case "LINK":
		// 🔴 Three answers and no snooze, for the same reason as INVITE — and
		// one more: the request expires in ten minutes, so «позже» would be a
		// button that quietly does nothing.
		//
		// Two rows, because «❌ Это не я» must not sit beside the two answers
		// that both say yes. Denis's rule from the four passes on 17.09 is that
		// every correction goes toward fewer steps, but a destructive answer
		// next to an affirmative one is not fewer steps, it is a misclick.
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "Оставить аккаунт сайта", "Keep the site account"), EncodeCallback(codeKeepSite, reminderID)),
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "Оставить Telegram", "Keep Telegram"), EncodeCallback(codeKeepTg, reminderID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "❌ Это не я", "❌ This is not me"), EncodeCallback(codeDecline, reminderID)),
			),
		)
		return &kb

	case "DIGEST":
		return nil
	default:
		// Events, and anything new that has not been given its own buttons: an
		// acknowledgement and a postponement are meaningful for any single item.
		row = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "👌 Понятно", "👌 Got it"), EncodeCallback(codeAck, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ 10 мин", "⏰ 10 min"), EncodeCallback(codeSnooze, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ Час", "⏰ An hour"), EncodeCallback(codeSnoozeHour, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(l, "⏰ Своё", "⏰ Custom"), EncodeCallback(codeSnoozeAsk, reminderID)),
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
	return &kb
}

// KeyboardFits reports whether every button in a keyboard is inside Telegram's
// callback_data cap.
//
// Telegram rejects the WHOLE message when one button is over the limit, so a
// notification with an oversized button is a notification that never arrives.
// Checking the real buttons rather than a representative one means a keyboard
// that stops carrying the button we happened to sample cannot slip past.
func KeyboardFits(kb *tgbotapi.InlineKeyboardMarkup) bool {
	if kb == nil {
		return false
	}
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if !FitsCallbackLimit(*b.CallbackData) {
				return false
			}
		}
	}
	return true
}

// CallbackDataLimit is Telegram's hard cap on callback_data.
const CallbackDataLimit = 64

// FitsCallbackLimit reports whether an encoded payload is short enough to send.
func FitsCallbackLimit(data string) bool {
	return len([]byte(data)) <= CallbackDataLimit
}

// ParseMinutes is used by callers that carry an explicit figure in the payload.
// Kept for the day a "+1 hour" button appears; unknown input falls back to the
// default rather than to zero, which would mean "immediately".
func ParseMinutes(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return SnoozeMinutes
	}
	return n
}

// ActionReply is what the chat says after a button press succeeded.
//
// 🔴 It exists because of what happened on 17.08, the first time an INVITE
// notification was pressed for real. The API answered 200 to every press and
// the membership flipped to active — and the bot said NOTHING, because the
// switch that produces this text had cases for snooze, done and ack and none
// for accept or decline. From the chat it looked broken, so Denis pressed
// seven times.
//
// The action worked and the screen stayed silent: the same defect as an event
// colour that is stored and never painted, one layer out. A silent success is
// indistinguishable from a failure, and the user's only move is to press again.
//
// A pure function, so ActionsAllHaveReplies can hold it against the buttons the
// keyboards actually emit. A switch buried in a handler that needs a live bot
// could not be tested, which is why the gap shipped.
func ActionReply(lang i18n.Lang, action string, minutes int) string {
	switch action {
	case ActionSnooze:
		// 🔴 The interval comes from the press, not from a constant. This line
		// said «через 10 минут» for every snooze — including the hour
		// button added the same day — which is a confident sentence about
		// something that did not happen.
		return i18n.T(lang,
			"⏰ Напомню через "+HumanMinutes(lang, minutes)+".",
			"⏰ I will remind you in "+HumanMinutes(lang, minutes)+".")
	case ActionSnoozeAsk:
		// Answered by the chat, which asks the question; nothing to report yet.
		return ""
	case ActionDone:
		return i18n.T(lang, "✅ Готово.", "✅ Done.")
	case ActionAck:
		return "👌"
	case ActionAccept:
		return i18n.T(lang, "✅ Календарь добавлен, он появится в списке.", "✅ Calendar added; it will show up in the list.")
	case ActionDecline:
		return i18n.T(lang, "Приглашение отклонено.", "Invitation declined.")
	case ActionKeepSite, ActionKeepTg:
		// 🔴 Neutral on purpose, and not empty. What happens next depends on the
		// API's answer — a second question about calendars, or a finished merge
		// — so naming either here would be a claim made before the fact. But
		// TestEveryButtonHasAReply is right that silence reads as failure, so
		// this says the one thing that is true in both branches, and the handler
		// says the outcome once it knows it.
		return i18n.T(lang, "Принял.", "Got it.")
	case ActionCalMerge, ActionCalBoth:
		return ""
	}
	return ""
}

// KnownSourceKinds is every source_kind the API can send. Kept here so the test
// that checks "every button has a reply" iterates the real set rather than a
// list written beside it, which would agree with itself by construction.
var KnownSourceKinds = []string{"EVENT", "TASK", "DIGEST", "INVITE", "LINK"}

// humanMinutes says an interval the way a person would.
//
// ⚠ Russian needs the plural form and English does not, so each language gets
// its own sentence rather than a shared template with a number poked into it.
func HumanMinutes(lang i18n.Lang, m int) string {
	switch {
	case m >= 1440 && m%1440 == 0:
		d := m / 1440
		return i18n.T(lang,
			itoa(d)+" "+ruPlural(d, "день", "дня", "дней"),
			itoa(d)+" days")
	case m >= 60 && m%60 == 0:
		h := m / 60
		return i18n.T(lang,
			itoa(h)+" "+ruPlural(h, "час", "часа", "часов"),
			itoa(h)+" hours")
	default:
		return i18n.T(lang,
			itoa(m)+" "+ruPlural(m, "минуту", "минуты", "минут"),
			itoa(m)+" minutes")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// ruPlural picks the Russian form for a count: 1 минута, 3 минуты, 5 минут.
func ruPlural(n int, one, few, many string) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return one
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return few
	default:
		return many
	}
}
