package config

import "os"

type Config struct {
	TelegramToken string
	APIBase       string
	BotPort       string
	Timezone      string
}

func Load() Config {
	c := Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		APIBase:       os.Getenv("API_BASE"),
		BotPort:       os.Getenv("BOT_PORT"),
		Timezone:      os.Getenv("TIMEZONE"),
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
