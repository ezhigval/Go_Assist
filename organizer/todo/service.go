package todo

import (
	"context"
	"fmt"
	"organizer"
	"sync"
	"time"
)

// Service реализует TodoService
type Service struct {
	storage organizer.Storage
	mu      sync.RWMutex
	local   map[string]*organizer.Todo // Кэш для демонстрации (заменится на storage)
}

// NewService создаёт сервис задач
func NewService(storage organizer.Storage) *Service {
	return &Service{storage: storage, local: make(map[string]*organizer.Todo)}
}

func (s *Service) Create(ctx context.Context, t *organizer.Todo) error {
	if t.ID == "" {
		t.ID = fmt.Sprintf("todo_%d", time.Now().UnixNano())
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	s.mu.Lock()
	s.local[t.ID] = t
	s.mu.Unlock()

	// STUB: Persistence requires storage.Save(t) after local cache when non-nil Storage; save scope/tags in Meta when schema fields missing.
	return nil
}

func (s *Service) Get(id string) (*organizer.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.local[id]
	if !ok {
		return nil, fmt.Errorf("todo %s not found", id)
	}
	return t, nil
}

func (s *Service) List() ([]organizer.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]organizer.Todo, 0, len(s.local))
	for _, t := range s.local {
		list = append(list, *t)
	}
	return list, nil
}

func (s *Service) Update(id string, t *organizer.Todo) error {
	t.UpdatedAt = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.local[id] = t
	return nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.local, id)
	return nil
}
