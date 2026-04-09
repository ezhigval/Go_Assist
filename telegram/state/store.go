package state

import "context"

// Session хранит контекст диалога пользователя
type Session struct {
	Key     string                 `json:"key"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// NewSession создаёт новую сессию
func NewSession(key string) Session {
	return Session{Key: key, Payload: make(map[string]interface{})}
}

// Store — интерфейс хранилища состояний (поддержка in-memory, redis, db)
type Store interface {
	Get(ctx context.Context, chatID int64) Session
	Set(ctx context.Context, chatID int64, session Session) error
	Clear(ctx context.Context, chatID int64) error
}
