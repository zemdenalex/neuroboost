package config

import "os"

type Config struct {
	TelegramToken string
	APIBase       string
	BotPort       string
	Timezone      string
	ProxyURL      string // SOCKS5 or HTTP proxy for Telegram API (e.g. socks5://host:1080)
	// ServiceToken authenticates the notifier against /api/svc. No default:
	// unset means notifications stay off rather than failing every minute.
	ServiceToken string
	// BotUsername is this bot's @name, used to build invite links
	// (https://t.me/<username>?start=inv_<token>).
	//
	// 🔴 Normally EMPTY and unused: the bot learns its own name from getMe at
	// startup (bot.Self.UserName), which cannot disagree with the token it is
	// running on. This field is only an override for a test or an odd proxy
	// setup. Hardcoding a name would be the real hazard — dev and prod are two
	// different bots (@NeuroBoost_dev_bot, @NeuroBoost_assistant_bot), and a
	// baked-in one would send every dev invitation to the production bot, where
	// the calendar does not exist.
	BotUsername string
}

func Load() Config {
	c := Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		APIBase:       os.Getenv("API_BASE"),
		BotPort:       os.Getenv("BOT_PORT"),
		Timezone:      os.Getenv("TIMEZONE"),
		ProxyURL:      os.Getenv("TELEGRAM_PROXY"),
		ServiceToken:  os.Getenv("SERVICE_TOKEN"),
		BotUsername:   os.Getenv("BOT_USERNAME"),
	}
	if c.APIBase == "" {
		c.APIBase = "http://localhost:8080"
	}
	if c.BotPort == "" {
		c.BotPort = "3002"
	}
	if c.Timezone == "" {
		c.Timezone = "Europe/Moscow"
	}
	return c
}
