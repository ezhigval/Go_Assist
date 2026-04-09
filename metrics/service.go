package metrics

import (
	"context"
	"log"
	"sync"

	"modulr/events"
)

var _ MetricsAPI = (*Service)(nil)

// Service пассивный слушатель событий для счётчиков и отчётности (аномалии/тренды — поверх тех же данных).
type Service struct {
	cfg     Config
	bus     *events.Bus
	idem    events.IdempotencyStore
	mu      sync.Mutex
	counts  map[string]int64
	started bool
}

// NewService создаёт metrics-сервис.
func NewService(cfg Config, bus *events.Bus, idem events.IdempotencyStore) *Service {
	return &Service{
		cfg:    cfg,
		bus:    bus,
		idem:   idem,
		counts: make(map[string]int64),
	}
}

// Start подписывается на все события шины.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.bus != nil {
		s.bus.SubscribeAll(s.onEvent)
	}
	s.started = true
	log.Println("metrics: module started")
	return nil
}

func (s *Service) onEvent(evt events.Event) {
	if s.cfg.DedupeByEventID && s.idem != nil && evt.ID != "" {
		key := "metrics-" + evt.ID
		if s.idem.Seen(key) {
			return
		}
		s.idem.MarkSeen(key)
	}
	s.mu.Lock()
	s.counts[string(evt.Name)]++
	s.counts["__total__"]++
	s.mu.Unlock()
}

// Counts снимок агрегатов (сырьё для дашбордов и DetectAnomaly в будущем).
func (s *Service) Counts() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64, len(s.counts))
	for k, v := range s.counts {
		out[k] = v
	}
	return out
}

// Stop останавливает модуль.
func (s *Service) Stop() error {
	log.Println("metrics: module stopped")
	return nil
}
