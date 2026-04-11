package auth

import (
	"context"
	"fmt"
	"sync"
)

// MemorySessionStore in-memory сессии для dev/тестов.
type MemorySessionStore struct {
	mu   sync.RWMutex
	data map[string]*Session
}

// NewMemorySessionStore создаёт хранилище.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{data: make(map[string]*Session)}
}

// Put сохраняет сессию по токену.
func (m *MemorySessionStore) Put(_ context.Context, token string, s *Session) error {
	if token == "" || s == nil {
		return fmt.Errorf("auth: invalid session put")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[token] = cloneSession(s)
	return nil
}

// Get возвращает сессию.
func (m *MemorySessionStore) Get(_ context.Context, token string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.data[token]
	if !ok {
		return nil, fmt.Errorf("auth: session not found")
	}
	return cloneSession(s), nil
}

// Delete удаляет сессию.
func (m *MemorySessionStore) Delete(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, token)
	return nil
}

func cloneSession(src *Session) *Session {
	if src == nil {
		return nil
	}
	dst := *src
	if len(src.Roles) != 0 {
		dst.Roles = append([]Role(nil), src.Roles...)
	}
	if len(src.AllowedScopes) != 0 {
		dst.AllowedScopes = append([]string(nil), src.AllowedScopes...)
	}
	if len(src.Meta) != 0 {
		dst.Meta = make(map[string]any, len(src.Meta))
		for k, v := range src.Meta {
			dst.Meta[k] = v
		}
	}
	return &dst
}
