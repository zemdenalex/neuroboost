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
	// 🔴 Read from the environment, never hardcoded: dev and prod are two
	// different bots on the same host (@NeuroBoost_dev_bot and
	// @NeuroBoost_assistant_bot), and a baked-in name would send every dev
	// invitation to the production bot — where the calendar does not exist.
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
