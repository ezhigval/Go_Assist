package telegram

import "os"

// Config хранит параметры запуска бота
type Config struct {
	Token          string
	Mode           string // "polling" | "webhook"
	WebhookURL     string
	WebhookPath    string
	ServerPort     string
	AllowedUpdates []string
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
