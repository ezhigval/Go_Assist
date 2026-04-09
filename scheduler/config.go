package scheduler

import (
	"os"
	"strconv"
	"time"
)

// Config параметры планировщика.
type Config struct {
	ReminderLead    time.Duration // за сколько до события слать напоминание
	DefaultRetryMax int
	RetryBackoff    time.Duration
}

// LoadConfig из окружения.
func LoadConfig() Config {
	lead := 15 * time.Minute
	if v := os.Getenv("SCHEDULER_REMINDER_LEAD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			lead = d
		}
	}
	retry := 3
	if v := os.Getenv("SCHEDULER_RETRY_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			retry = n
		}
	}
	backoff := 2 * time.Second
	if v := os.Getenv("SCHEDULER_RETRY_BACKOFF"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			backoff = d
		}
	}
	return Config{
		ReminderLead:    lead,
		DefaultRetryMax: retry,
		RetryBackoff:    backoff,
	}
}
