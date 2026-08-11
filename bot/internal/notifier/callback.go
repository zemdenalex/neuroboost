package notifier

import (
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
)

// Action names as the API expects them. Kept separate from the wire codes so
// shortening the wire format can never silently rename an API action.
const (
	ActionAck    = "ack"
	ActionSnooze = "snooze"
	ActionDone   = "done"
)

// SnoozeMinutes is what the "later" button asks for.
const SnoozeMinutes = 10

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
	case codeDone:
		return Callback{Action: ActionDone, ReminderID: id}, true
	}
	return Callback{}, false
}

// Keyboard builds the buttons for one notification.
//
// A digest gets none: it is a summary of several things, so "done" and "later"
// have no single subject to act on.
func Keyboard(sourceKind, reminderID string) *tgbotapi.InlineKeyboardMarkup {
	var row []tgbotapi.InlineKeyboardButton

	switch strings.ToUpper(sourceKind) {
	case "TASK":
		row = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово", EncodeCallback(codeDone, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ +10 мин", EncodeCallback(codeSnooze, reminderID)),
		}
	case "DIGEST":
		return nil
	default:
		// Events, and anything new that has not been given its own buttons: an
		// acknowledgement and a postponement are meaningful for any single item.
		row = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👌 Понятно", EncodeCallback(codeAck, reminderID)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ +10 мин", EncodeCallback(codeSnooze, reminderID)),
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
	return &kb
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
