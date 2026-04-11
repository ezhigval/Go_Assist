package telegram

import (
	"os"
	"strconv"
	"time"
)

// Config хранит параметры запуска бота
type Config struct {
	Token          string
	Mode           string // "polling" | "webhook"
	WebhookURL     string
	WebhookPath    string
	ServerPort     string
	AllowedUpdates []string
	StateStore     string
	DefaultScope   string
	RuntimeTimeout time.Duration
}

// LoadConfig считывает настройки из переменных окружения
func LoadConfig() Config {
	return Config{
		Token:          getEnv("TELEGRAM_TOKEN", ""),
		Mode:           getEnv("BOT_MODE", "polling"),
		WebhookURL:     getEnv("WEBHOOK_URL", ""),
		WebhookPath:    getEnv("WEBHOOK_PATH", "/webhook"),
		ServerPort:     getEnv("PORT", "8080"),
		AllowedUpdates: []string{"message", "callback_query"},
		StateStore:     getEnv("TELEGRAM_STATE_STORE", "memory"),
		DefaultScope:   getEnv("TELEGRAM_DEFAULT_SCOPE", "personal"),
		RuntimeTimeout: time.Duration(getEnvInt("TELEGRAM_RUNTIME_TIMEOUT_MS", 3500)) * time.Millisecond,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
