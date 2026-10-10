// The inbox bot: a separate Telegram bot, owner-only, that saves whatever
// Denis sends it to /api/inbox. Spec: docs/team/architecture/V003-20261010-arc-inbox-bot.md.
//
// Its env names are its own (INBOX_*), never TELEGRAM_BOT_TOKEN: a .env copied
// from the main bot must not start a second instance on the main bot's token —
// two pollers on one token make delivery flicker (gotcha 15).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/zemdenalex/neuroboost-bot/inbox"
)

func main() {
	token := os.Getenv("INBOX_BOT_TOKEN")
	owner, _ := strconv.ParseInt(os.Getenv("INBOX_OWNER_TG_ID"), 10, 64)
	apiToken := os.Getenv("INBOX_API_TOKEN")
	apiBase := strings.TrimRight(os.Getenv("API_BASE"), "/")
	if token == "" || owner == 0 || apiToken == "" || apiBase == "" {
		log.Fatal("INBOX_BOT_TOKEN, INBOX_OWNER_TG_ID, INBOX_API_TOKEN and API_BASE are required")
	}
	// 950 KB: the host nginx in front of the API cuts bodies at 1 MB today.
	maxBytes := int64(950_000)
	if v, err := strconv.ParseInt(os.Getenv("INBOX_MAX_UPLOAD_BYTES"), 10, 64); err == nil && v > 0 {
		maxBytes = v
	}
	projects := []string{"neuroboost", "avocar", "archifex", "harness", "graph"}
	if v := strings.TrimSpace(os.Getenv("INBOX_PROJECTS")); v != "" {
		projects = strings.Split(v, ",")
	}

	s := &inbox.Service{Owner: owner, Token: token, API: &inbox.API{Base: apiBase, Token: apiToken},
		Projects: projects, MaxBytes: maxBytes}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	b, err := bot.New(token,
		bot.WithDefaultHandler(s.Handle),
		bot.WithAllowedUpdates(bot.AllowedUpdates{
			models.AllowedUpdateMessage, models.AllowedUpdateCallbackQuery,
			models.AllowedUpdateInlineQuery, models.AllowedUpdateChosenInlineResult,
		}),
		// The library's own error log prints a failed getUpdates with its URL,
		// and the URL is …/bot<TOKEN>/getUpdates.
		bot.WithErrorsHandler(func(err error) { log.Printf("telegram: %s", inbox.Redact(err.Error(), token)) }),
	)
	if err != nil {
		log.Fatalf("bot: %s", inbox.Redact(err.Error(), token))
	}
	s.Bind(b)
	me, err := b.GetMe(ctx)
	if err != nil {
		log.Fatalf("getMe: %s", inbox.Redact(err.Error(), token))
	}
	log.Printf("Bot started as @%s, owner %d, api %s", me.Username, owner, apiBase)
	b.Start(ctx)
}
