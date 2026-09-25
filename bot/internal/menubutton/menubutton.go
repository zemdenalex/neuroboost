// Package menubutton installs the chat menu button that opens NeuroBoost as a
// Telegram Mini App (docs/tasks-mini-app.md, MA9).
//
// go-telegram-bot-api v5.5.1 predates setChatMenuButton (Bot API 6.0), so the
// request goes through MakeRequest with the documented JSON shape.
package menubutton

import (
	"encoding/json"
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Requester is the one method of *tgbotapi.BotAPI this package needs.
type Requester interface {
	MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error)
}

// Set makes the default menu button (every chat without its own) open url as
// a Mini App. Telegram opens only https Mini Apps, so anything else is refused
// before a request is sent.
func Set(bot Requester, url, text string) error {
	if !strings.HasPrefix(url, "https://") {
		return errors.New("menu button: Mini App URL must be https")
	}
	button, err := json.Marshal(map[string]any{
		"type":    "web_app",
		"text":    text,
		"web_app": map[string]string{"url": url},
	})
	if err != nil {
		return err
	}
	_, err = bot.MakeRequest("setChatMenuButton", tgbotapi.Params{"menu_button": string(button)})
	return err
}
