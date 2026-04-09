package contacts

import (
	"context"
	"fmt"
	"organizer"
	"sync"
	"time"
)

// Service реализует ContactService
type Service struct {
	storage organizer.Storage
	mu      sync.RWMutex
	local   map[string]*organizer.Contact
}

// NewService создаёт сервис контактов
func NewService(storage organizer.Storage) *Service {
	return &Service{storage: storage, local: make(map[string]*organizer.Contact)}
}

func (s *Service) Create(ctx context.Context, c *organizer.Contact) error {
	if c.ID == "" {
		c.ID = fmt.Sprintf("contact_%d", time.Now().UnixNano())
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	s.mu.Lock()
	s.local[c.ID] = c
	s.mu.Unlock()

	// STUB: External sync requires CardDAV/Google People adapter via SyncSource interface called from worker with ctx and rate limiting.
	return nil
}

func (s *Service) Get(id string) (*organizer.Contact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.local[id]
	if !ok {
		return nil, fmt.Errorf("contact %s not found", id)
	}
	return c, nil
}

func (s *Service) List() ([]organizer.Contact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]organizer.Contact, 0, len(s.local))
	for _, c := range s.local {
		list = append(list, *c)
	}
	return list, nil
}
