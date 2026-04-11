package app

import (
	"context"
	"time"
)

// JournalRecord запись для внешнего trace/journal sink.
type JournalRecord struct {
	TraceID   string
	ChatID    int64
	Scope     string
	EventName string
	Status    string
	Source    string
	Payload   map[string]any
	Metadata  map[string]any
	CreatedAt time.Time
}

// EventJournal сохраняет trace-связанные runtime/transport события.
type EventJournal interface {
	WriteEvent(ctx context.Context, record JournalRecord) error
}

// RuntimeOption конфигурирует дополнительные runtime-зависимости.
type RuntimeOption func(*Runtime)

// WithEventJournal подключает внешний sink для входящих сообщений и outcome/fallback.
func WithEventJournal(journal EventJournal) RuntimeOption {
	return func(r *Runtime) {
		r.journal = journal
	}
}
