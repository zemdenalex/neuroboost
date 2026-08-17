package config

import (
	"os"
)

type Config struct {
	DatabaseURL       string
	JWTSecret         string
	FrontendURL       string
	TelegramBotToken  string
	TelegramNotifyBot string
	// ServiceToken guards the /api/svc endpoints the notifier bot calls. There
	// is deliberately no default: a default service token is a backdoor, and
	// an unset one closes the endpoints rather than opening them.
	ServiceToken string
	Environment  string
}

func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://neuroboost:neuroboost@localhost:5432/neuroboost?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:5173"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramNotifyBot: os.Getenv("TELEGRAM_NOTIFICATION_BOT_TOKEN"),
		ServiceToken:      os.Getenv("SERVICE_TOKEN"),
		Environment:       getEnv("NODE_ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
