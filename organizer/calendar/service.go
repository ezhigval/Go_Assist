package calendar

import (
	"context"
	"fmt"
	"organizer"
	"sync"
	"time"
)

// Service реализует CalendarService
type Service struct {
	storage organizer.Storage
	bus     *organizer.EventBus
	mu      sync.RWMutex
	local   map[string]*organizer.CalendarEvent
}

// NewService создаёт сервис календаря
func NewService(storage organizer.Storage, bus *organizer.EventBus) *Service {
	return &Service{storage: storage, bus: bus, local: make(map[string]*organizer.CalendarEvent)}
}

func (s *Service) Create(ctx context.Context, e *organizer.CalendarEvent) error {
	if e.ID == "" {
		e.ID = fmt.Sprintf("cal_%d", time.Now().UnixNano())
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()

	s.mu.Lock()
	s.local[e.ID] = e
	s.mu.Unlock()

	// Публикуем событие для кросс-модульных связей
	s.bus.Publish(organizer.Event{
		Name:    organizer.EventEventCreated,
		Payload: e,
		Source:  "calendar",
	})
	return nil
}

func (s *Service) Get(id string) (*organizer.CalendarEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.local[id]
	if !ok {
		return nil, fmt.Errorf("event %s not found", id)
	}
	return e, nil
}

func (s *Service) List() ([]organizer.CalendarEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]organizer.CalendarEvent, 0, len(s.local))
	for _, e := range s.local {
		list = append(list, *e)
	}
	return list, nil
}
