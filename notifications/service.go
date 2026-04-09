package notifications

import (
	"context"
	"fmt"
	"log"
	"sync"

	"modulr/events"
)

// Service подписывается на *.created и *.due, не блокирует шину.
type Service struct {
	cfg  Config
	bus  *events.Bus
	sink Sink

	mu      sync.Mutex
	started bool
}

// NewService создаёт модуль уведомлений.
func NewService(cfg Config, bus *events.Bus, sink Sink) *Service {
	if sink == nil {
		sink = LogSink{}
	}
	return &Service{cfg: cfg, bus: bus, sink: sink}
}

// Start регистрирует суффиксные подписки.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.bus == nil {
		return fmt.Errorf("notifications: bus is nil")
	}
	s.bus.SubscribeSuffix(".created", s.dispatch)
	s.bus.SubscribeSuffix(".due", s.dispatch)
	s.started = true
	log.Println("notifications: module started")
	return nil
}

func (s *Service) dispatch(evt events.Event) {
	// не шлём уведомления о служебных событиях аналитики/ai
	switch evt.Name {
	case events.V1AISuggestion, events.V1SystemStartup:
		return
	}
	target := ""
	if evt.Context != nil {
		if v, ok := evt.Context["chat_id"].(int64); ok {
			target = fmt.Sprintf("chat:%d", v)
		}
		if v, ok := evt.Context["user_id"].(string); ok && target == "" {
			target = "user:" + v
		}
	}
	n := Notification{
		Channel: s.cfg.ChannelDefault,
		Target:  target,
		Title:   fmt.Sprintf("Event %s", evt.Name),
		Body:    fmt.Sprintf("from %s", evt.Source),
		TraceID: evt.TraceID,
	}
	go func() {
		ctx := context.Background()
		if err := s.sink.Send(ctx, n); err != nil {
			log.Printf("notifications: sink error: %v", err)
		}
	}()
}

// Stop заглушка.
func (s *Service) Stop() error {
	log.Println("notifications: module stopped")
	return nil
}
