package aiengine

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config управляет gated AI provider integration для core/aiengine.
type Config struct {
	Provider          string
	ProviderBaseURL   string
	ProviderTimeout   time.Duration
	AllowStubFallback bool
}

// LoadConfig читает настройки из окружения. Дефолт — полностью локальный stub.
func LoadConfig() Config {
	timeout := 2500 * time.Millisecond
	if v := os.Getenv("AI_PROVIDER_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}

	return Config{
		Provider:          strings.ToLower(strings.TrimSpace(getEnv("AI_PROVIDER", "stub"))),
		ProviderBaseURL:   strings.TrimRight(getEnv("AI_PROVIDER_BASE_URL", "http://127.0.0.1:8000"), "/"),
		ProviderTimeout:   timeout,
		AllowStubFallback: getEnvBool("AI_ALLOW_STUB_FALLBACK", true),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}
