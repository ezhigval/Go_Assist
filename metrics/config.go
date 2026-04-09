package metrics

import "os"

// Config модуля метрик.
type Config struct {
	DedupeByEventID bool
}

// LoadConfig из окружения.
func LoadConfig() Config {
	return Config{DedupeByEventID: os.Getenv("METRICS_DEDUPE") == "true"}
}
