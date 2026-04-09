package state

import (
	"context"
	"sync"
)

// MemoryStore — реализация Store для разработки/локальных тестов
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[int64]Session
}

// NewMemoryStore создаёт хранилище в памяти
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[int64]Session),
	}
}

func (m *MemoryStore) Get(_ context.Context, chatID int64) Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[chatID]
}

func (m *MemoryStore) Set(_ context.Context, chatID int64, session Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[chatID] = session
	return nil
}

func (m *MemoryStore) Clear(_ context.Context, chatID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chatID)
	return nil
}
