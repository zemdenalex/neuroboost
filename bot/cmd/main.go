package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/handlers"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

func main() {
	cfg := config.Load()
	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	// Create bot — with optional proxy for Telegram API
	// Russian hosting blocks Telegram IPv4. Set TELEGRAM_PROXY=http://euro-server:8888 to route through proxy.
	var bot *tgbotapi.BotAPI
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			log.Fatalf("Invalid TELEGRAM_PROXY: %v", err)
		}
		httpClient := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		}
		bot, err = tgbotapi.NewBotAPIWithClient(cfg.TelegramToken, tgbotapi.APIEndpoint, httpClient)
		if err != nil {
			log.Fatalf("Failed to create bot with proxy: %s", logsafe.Redact(err))
		}
		log.Printf("Using proxy: %s", proxyURL.Host)
	} else {
		var err error
		bot, err = tgbotapi.NewBotAPI(cfg.TelegramToken)
		if err != nil {
			log.Fatal("Failed to create bot (set TELEGRAM_PROXY if Telegram is blocked): ", logsafe.Redact(err))
		}
	}
	log.Printf("Bot authorized as @%s", bot.Self.UserName)

	apiClient := api.NewClient(cfg.APIBase)
	store := state.NewStore()
	h := handlers.New(bot, apiClient, store, cfg)

	// Health server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})
		log.Printf("Health server on :%s", cfg.BotPort)
		http.ListenAndServe(":"+cfg.BotPort, mux)
	}()

	// Polling
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		bot.StopReceivingUpdates()
		cancel()
	}()

	// Reminder delivery shares this process and this ctx, so it stops with the
	// bot on SIGTERM rather than outliving it.
	go notifier.Start(ctx, bot, apiClient, cfg.ServiceToken, time.Minute)

	log.Println("Bot started, listening for updates...")
	for {
		select {
		case update := <-updates:
			// 🔴 One update must never be able to kill the bot for everyone.
			//
			// This loop is the whole process: a panic in any handler unwinds
			// through here and terminates it, and the bot is deployed BY HAND
			// off a separate host, so "it restarts" is not a comfort — someone
			// has to notice first. gotcha 9 in CLAUDE.md says every background
			// goroutine needs a recover; this loop is the goroutine whose panic
			// takes down everything, and until 14.08 it was the one place
			// without one.
			//
			// A nil CallbackQuery.Message was the known way in (Telegram omits
			// it for old messages) and is now checked at its own door too. This
			// guard is for the ones nobody has found yet.
			handleUpdate(h, update)
		case <-ctx.Done():
			log.Println("Bot stopped")
			return
		}
	}
}

// handleUpdate routes one update and contains any panic it causes.
//
// Deliberately NOT logging the panic value through a plain %v: a panic raised
// from inside the Telegram client can carry an error whose message holds the
// bot token.
func handleUpdate(h *handlers.Handler, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic while handling update %d: %s\n%s",
				update.UpdateID, logsafe.String(fmt.Sprint(r)), debug.Stack())
		}
	}()

	if update.Message != nil {
		h.HandleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		h.HandleCallback(update.CallbackQuery)
	}
}
