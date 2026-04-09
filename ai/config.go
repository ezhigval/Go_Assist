package ai

import (
	"os"
	"strconv"
)

// Config модуля AI.
type Config struct {
	MinConfidence float64
	MaxBuffer     int
}

// LoadConfig из окружения.
func LoadConfig() Config {
	mc := 0.6
	if v := os.Getenv("AI_MIN_CONFIDENCE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			mc = f
		}
	}
	maxB := 32
	if v := os.Getenv("AI_CONTEXT_BUFFER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxB = n
		}
	}
	return Config{MinConfidence: mc, MaxBuffer: maxB}
}
