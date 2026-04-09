package auth

import (
	"os"
	"time"
)

// Config параметры модуля auth.
type Config struct {
	SessionTTL       time.Duration
	EnableEventAudit bool
}

// LoadConfig из окружения.
func LoadConfig() Config {
	ttl := 24 * time.Hour
	if v := os.Getenv("AUTH_SESSION_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	return Config{
		SessionTTL:       ttl,
		EnableEventAudit: os.Getenv("AUTH_EVENT_AUDIT") == "true",
	}
}
