package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"modulr/events"
)

// Service слушает шину, строит краткий контекст, публикует v1.ai.suggestion.
type Service struct {
	cfg     Config
	bus     *events.Bus
	gateway Gateway

	mu          sync.Mutex
	buffer      []string
	lastSuggest time.Time
	started     bool
}

// NewService создаёт AI-модуль.
func NewService(cfg Config, bus *events.Bus, gw Gateway) *Service {
	if gw == nil {
		gw = StubGateway{}
	}
	if cfg.MaxBuffer <= 0 {
		cfg.MaxBuffer = 32
	}
	return &Service{cfg: cfg, bus: bus, gateway: gw, buffer: make([]string, 0, cfg.MaxBuffer)}
}

// Start подписывается на все события.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.bus == nil {
		return fmt.Errorf("ai: bus is nil")
	}
	s.bus.SubscribeAll(s.onEvent)
	s.started = true
	log.Println("ai: module started")
	return nil
}

func (s *Service) onEvent(evt events.Event) {
	// Не реагируем на свои предложения — иначе лавина на шине.
	if evt.Name == events.V1AISuggestion {
		return
	}
	line := string(evt.Name) + "@" + evt.Source
	s.mu.Lock()
	if len(s.buffer) >= s.cfg.MaxBuffer {
		s.buffer = s.buffer[1:]
	}
	s.buffer = append(s.buffer, line)
	if len(s.buffer) < 3 {
		s.mu.Unlock()
		return
	}
	if time.Since(s.lastSuggest) < 400*time.Millisecond {
		s.mu.Unlock()
		return
	}
	s.lastSuggest = time.Now()
	prompt := strings.Join(s.buffer, "; ")
	s.mu.Unlock()
	prompt = RedactPII(prompt)

	ctx := context.Background()
	text, conf, err := s.gateway.Complete(ctx, prompt)
	if err != nil {
		log.Printf("ai: gateway error: %v", err)
		return
	}
	if conf < s.cfg.MinConfidence {
		return
	}
	sug := Suggestion{
		TargetModule: "organizer",
		Action:       text,
		Reason:       "context_roll",
		Confidence:   conf,
		Payload:      map[string]any{"last_event": string(evt.Name)},
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    events.V1AISuggestion,
			Payload: sug,
			Source:  "ai",
			TraceID: evt.TraceID,
			Context: map[string]any{"confidence": conf},
		})
	}
}

// Stop заглушка.
func (s *Service) Stop() error {
	log.Println("ai: module stopped")
	return nil
}
