package notifications

import "os"

// Config модуля уведомлений.
type Config struct {
	ChannelDefault string
}

// LoadConfig из окружения.
func LoadConfig() Config {
	ch := os.Getenv("NOTIFY_DEFAULT_CHANNEL")
	if ch == "" {
		ch = "telegram"
	}
	return Config{ChannelDefault: ch}
}
