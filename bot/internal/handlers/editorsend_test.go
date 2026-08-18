package handlers

import (
	"errors"
	"testing"
)

// shouldSendNew holds the whole edit-or-send decision, because the surrounding
// method needs a bot, a store and a live API to run at all — and the decision
// is the only part that can be wrong.
func TestShouldSendNewWhenThereIsNoMessageToEdit(t *testing.T) {
	if !shouldSendNew(0, nil) {
		t.Error("messageID 0 means the press came from a reply button: there is nothing to edit")
	}
}

func TestShouldNotSendNewOnASuccessfulEdit(t *testing.T) {
	if shouldSendNew(42, nil) {
		t.Error("a successful edit must not also post a message — that is the message spam being removed")
	}
}

func TestShouldSendNewWhenTelegramRefusesTheEdit(t *testing.T) {
	// 🔴 Telegram forbids editing a message older than 48 hours. Staying silent
	// there is the worst outcome: the user pressed a button and nothing at all
	// happened, which reads as a dead bot.
	err := errors.New("Bad Request: message can't be edited")
	if !shouldSendNew(42, err) {
		t.Error("a refused edit must fall back to a new message, not silence")
	}
}

func TestNotModifiedIsNotAFailure(t *testing.T) {
	// Tapping the same page twice produces byte-identical text and markup.
	// That is not an error, and it must not spawn a duplicate message.
	err := errors.New("Bad Request: message is not modified")
	if shouldSendNew(42, err) {
		t.Error("an identical re-render must be swallowed, not answered with a new message")
	}
}
