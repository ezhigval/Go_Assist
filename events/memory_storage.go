package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryStorage потокобезопасная in-memory реализация Storage (тесты, демо).
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryStorage создаёт хранилище.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{data: make(map[string][]byte)}
}

// GetJSON читает значение по ключу.
func (m *MemoryStorage) GetJSON(_ context.Context, key string, dest any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	raw, ok := m.data[key]
	if !ok {
		return fmt.Errorf("events/memory: key not found: %s", key)
	}
	return json.Unmarshal(raw, dest)
}

// PutJSON сохраняет значение.
func (m *MemoryStorage) PutJSON(_ context.Context, key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = raw
	return nil
}

// Delete удаляет ключ.
func (m *MemoryStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// ListPrefix возвращает отсортированные ключи с префиксом.
func (m *MemoryStorage) ListPrefix(_ context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var keys []string
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}
