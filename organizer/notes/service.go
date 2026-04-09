package notes

import (
	"context"
	"fmt"
	"organizer"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Service реализует NoteService
type Service struct {
	storage organizer.Storage
	bus     *organizer.EventBus
	mu      sync.RWMutex
	local   map[string]*organizer.Note
}

// Regex-паттерны для извлечения данных (вынесены для переиспользования ИИ)
var (
	phoneRegex = regexp.MustCompile(`\+?\d[\d\s\-\(\)]{7,}\d`)
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
)

// NewService создаёт сервис заметок
func NewService(storage organizer.Storage, bus *organizer.EventBus) *Service {
	return &Service{storage: storage, bus: bus, local: make(map[string]*organizer.Note)}
}

func (s *Service) Create(ctx context.Context, n *organizer.Note) error {
	if n.ID == "" {
		n.ID = fmt.Sprintf("note_%d", time.Now().UnixNano())
	}
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	s.mu.Lock()
	s.local[n.ID] = n
	s.mu.Unlock()

	s.bus.Publish(organizer.Event{
		Name:    organizer.EventNoteSaved,
		Payload: n,
		Source:  "notes",
	})
	return nil
}

func (s *Service) Get(id string) (*organizer.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.local[id]
	if !ok {
		return nil, fmt.Errorf("note %s not found", id)
	}
	return n, nil
}

func (s *Service) List() ([]organizer.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]organizer.Note, 0, len(s.local))
	for _, n := range s.local {
		list = append(list, *n)
	}
	return list, nil
}

// ExtractPhones публичный метод для ИИ/других модулей
func (s *Service) ExtractPhones(text string) []string {
	matches := phoneRegex.FindAllString(text, -1)
	var res []string
	for _, m := range matches {
		res = append(res, strings.TrimSpace(m))
	}
	return res
}

// ExtractEmails публичный метод для ИИ/других модулей
func (s *Service) ExtractEmails(text string) []string {
	return emailRegex.FindAllString(text, -1)
}
