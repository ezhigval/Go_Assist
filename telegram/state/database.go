package state

import (
	"context"
	"fmt"
	"log"
)

// StoredSession запись сессии во внешнем хранилище.
type StoredSession struct {
	Key     string
	Payload map[string]interface{}
}

// SessionRepository минимальный контракт для DB-backed transport store.
type SessionRepository interface {
	EnsureChat(ctx context.Context, chatID int64) error
	GetSession(ctx context.Context, chatID int64) (StoredSession, error)
	SetSession(ctx context.Context, chatID int64, session StoredSession) error
	ClearSession(ctx context.Context, chatID int64) error
}

// DatabaseStore хранит состояние диалога во внешнем репозитории.
type DatabaseStore struct {
	repo SessionRepository
}

// NewDatabaseStore создаёт Store поверх внешнего репозитория сессий.
func NewDatabaseStore(repo SessionRepository) *DatabaseStore {
	return &DatabaseStore{repo: repo}
}

// Get возвращает текущее состояние; при ошибке чтения логирует и отдаёт пустую сессию.
func (s *DatabaseStore) Get(ctx context.Context, chatID int64) Session {
	if s == nil || s.repo == nil {
		return Session{}
	}

	stored, err := s.repo.GetSession(ctx, chatID)
	if err != nil {
		log.Printf("telegram/state: get session chat=%d: %v", chatID, err)
		return Session{}
	}
	return normalizeStoredSession(stored)
}

// Set сохраняет состояние в репозиторий.
func (s *DatabaseStore) Set(ctx context.Context, chatID int64, session Session) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("telegram/state: database repository is nil")
	}
	if err := s.repo.EnsureChat(ctx, chatID); err != nil {
		return err
	}
	return s.repo.SetSession(ctx, chatID, StoredSession{
		Key:     session.Key,
		Payload: clonePayload(session.Payload),
	})
}

// Clear удаляет состояние из репозитория.
func (s *DatabaseStore) Clear(ctx context.Context, chatID int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("telegram/state: database repository is nil")
	}
	return s.repo.ClearSession(ctx, chatID)
}

func normalizeStoredSession(stored StoredSession) Session {
	if stored.Key == "" || stored.Key == "idle" {
		return Session{}
	}
	return Session{
		Key:     stored.Key,
		Payload: clonePayload(stored.Payload),
	}
}

func clonePayload(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
