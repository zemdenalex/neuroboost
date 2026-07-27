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
