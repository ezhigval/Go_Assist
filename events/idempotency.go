package events

import "sync"

// IdempotencyStore — защита от повторной обработки одного и того же event.ID.
type IdempotencyStore interface {
	Seen(id string) bool
	MarkSeen(id string)
}

// MemoryIdempotency in-memory реализация (для dev/тестов; в проде — БД/Redis).
type MemoryIdempotency struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

// NewMemoryIdempotency создаёт хранилище обработанных id.
func NewMemoryIdempotency() *MemoryIdempotency {
	return &MemoryIdempotency{seen: make(map[string]struct{})}
}

// Seen возвращает true, если id уже обработан.
func (m *MemoryIdempotency) Seen(id string) bool {
	if id == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.seen[id]
	return ok
}

// MarkSeen помечает id как обработанный.
func (m *MemoryIdempotency) MarkSeen(id string) {
	if id == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen[id] = struct{}{}
}
