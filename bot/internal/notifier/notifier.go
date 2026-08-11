// Package notifier delivers the reminders the API computes. It is a goroutine
// inside the existing bot rather than a second service: both need the same bot
// token and the same Telegram client, and splitting them would mean two
// deploys and two secrets for one for-loop.
package notifier

import (
	"context"
	"log"
	"runtime/debug"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// Start polls for due notifications until ctx is cancelled.
func Start(ctx context.Context, bot *tgbotapi.BotAPI, client *api.Client, serviceToken string, interval time.Duration) {
	if serviceToken == "" {
		log.Println("notifier: SERVICE_TOKEN is not set, notifications disabled")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("notifier: polling every %s", interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("notifier: stopped")
			return
		case <-ticker.C:
			deliverBatch(bot, client, serviceToken)
		}
	}
}

// deliverBatch pulls one batch, sends it, and acks each row.
//
// The recover matters for the same reason it does in the API worker: this runs
// on a goroutine, so an unrecovered panic would kill the bot process and take
// every command down with it, not just one delivery round.
func deliverBatch(bot *tgbotapi.BotAPI, client *api.Client, serviceToken string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("notifier: panic in delivery round: %v\n%s", r, debug.Stack())
		}
	}()

	pending, err := client.PendingNotifications(serviceToken)
	if err != nil {
		// Ack nothing: the rows stay SENDING and the API reclaims them after
		// five minutes, so a pull failure costs a delay rather than a message.
		log.Printf("notifier: pull failed: %v", err)
		return
	}

	for _, n := range pending {
		msg := tgbotapi.NewMessage(n.TgID, n.Text)
		// Buttons are best-effort: if the payload could not be kept inside
		// Telegram's 64-byte callback_data cap, send the text alone rather than
		// lose the notification to a rejected message.
		if kb := Keyboard(n.SourceKind, n.ID); kb != nil && FitsCallbackLimit(EncodeCallback(codeSnooze, n.ID)) {
			msg.ReplyMarkup = *kb
		}
		_, sendErr := bot.Send(msg)

		delivered := sendErr == nil
		reason := ""
		if sendErr != nil {
			reason = sendErr.Error()
			log.Printf("notifier: send to %d failed: %v", n.TgID, sendErr)
		}
		// Ack even on failure. An un-acked row is retried forever, so a user
		// who blocked the bot would generate one failed send every minute
		// until somebody noticed.
		if err := client.AckNotification(serviceToken, n.ID, delivered, reason); err != nil {
			log.Printf("notifier: ack for %s failed: %v", n.ID, err)
		}
	}
}
