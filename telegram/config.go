package telegram

import (
	"modulr/events"
	"os"
	"strconv"
	"strings"
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
	AuthRequired   bool
	AuthAdminIDs   []int64
	AuthScopes     []string
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
		AuthRequired:   getEnv("TELEGRAM_AUTH_REQUIRED", "") == "true",
		AuthAdminIDs:   parseInt64CSV(getEnv("TELEGRAM_AUTH_ADMIN_IDS", "")),
		AuthScopes:     parseScopeCSV(getEnv("TELEGRAM_AUTH_ALLOWED_SCOPES", "")),
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

func parseInt64CSV(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func parseScopeCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		scope := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(part)))
		if scope == "" {
			continue
		}
		key := string(scope)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}
