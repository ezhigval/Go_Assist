package scheduler

import "time"

// CalendarHint минимальный контракт payload для v1.calendar.created (без импорта organizer).
type CalendarHint struct {
	ID    string
	Title string
	Start time.Time
}
