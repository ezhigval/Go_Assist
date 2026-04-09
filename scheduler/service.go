package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"modulr/events"
)

// Service реализует SchedulerAPI.
type Service struct {
	cfg  Config
	bus  *events.Bus
	idem events.IdempotencyStore

	mu       sync.Mutex
	timers   map[string]*time.Timer
	closed   bool
	started  bool
	stopOnce sync.Once
}

// NewService создаёт планировщик; idem опционален (идемпотентность триггеров по event.ID).
func NewService(cfg Config, bus *events.Bus, idem events.IdempotencyStore) *Service {
	return &Service{
		cfg:    cfg,
		bus:    bus,
		idem:   idem,
		timers: make(map[string]*time.Timer),
	}
}

func (s *Service) parseCalendarHint(p any) (CalendarHint, bool) {
	switch v := p.(type) {
	case CalendarHint:
		return v, v.Start.Unix() > 0
	case map[string]any:
		h := CalendarHint{}
		if id, ok := v["id"].(string); ok {
			h.ID = id
		}
		if title, ok := v["title"].(string); ok {
			h.Title = title
		}
		if ts, ok := v["start_time"].(time.Time); ok {
			h.Start = ts
			return h, true
		}
		if str, ok := v["start"].(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				h.Start = t
				return h, true
			}
		}
		return h, h.Start.Unix() > 0
	default:
		return CalendarHint{}, false
	}
}

// Start подписывается на v1.calendar.created и поднимает цикл отложенных задач.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.mu.Unlock()

	if s.bus != nil {
		s.bus.Subscribe(events.V1CalendarCreated, s.onCalendarCreated)
	}
	log.Println("scheduler: module started")
	return nil
}

func (s *Service) onCalendarCreated(evt events.Event) {
	if s.idem != nil && evt.ID != "" {
		if s.idem.Seen("sched-cal-" + evt.ID) {
			return
		}
		s.idem.MarkSeen("sched-cal-" + evt.ID)
	}
	hint, ok := s.parseCalendarHint(evt.Payload)
	if !ok {
		return
	}
	fireAt := hint.Start.Add(-s.cfg.ReminderLead)
	if fireAt.Before(time.Now()) {
		fireAt = time.Now().Add(time.Second)
	}
	_, err := s.ScheduleAt(context.Background(), fireAt, events.V1SchedulerTrigger, map[string]any{
		"calendar_id": hint.ID,
		"title":       hint.Title,
		"kind":        "reminder",
	})
	if err != nil {
		log.Printf("scheduler: schedule reminder failed: %v", err)
	}
}

// ScheduleAt планирует публикацию target на fireAt (через шину).
func (s *Service) ScheduleAt(ctx context.Context, fireAt time.Time, target events.Name, payload any) (string, error) {
	if s.bus == nil {
		return "", fmt.Errorf("scheduler: bus is nil")
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", fmt.Errorf("scheduler: stopped")
	}
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	d := time.Until(fireAt)
	if d < 0 {
		d = 0
	}
	t := time.AfterFunc(d, func() {
		tid := events.TraceIDFromContext(ctx)
		s.bus.Publish(events.Event{
			ID:      jobID,
			Name:    target,
			Payload: payload,
			Source:  "scheduler",
			TraceID: tid,
			Context: map[string]any{"scheduled_for": fireAt.UTC().Format(time.RFC3339)},
		})
		s.mu.Lock()
		delete(s.timers, jobID)
		s.mu.Unlock()
	})
	s.timers[jobID] = t
	s.mu.Unlock()
	return jobID, nil
}

// Cancel отменяет задачу по id.
func (s *Service) Cancel(jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.timers[jobID]
	if !ok {
		return false
	}
	t.Stop()
	delete(s.timers, jobID)
	return true
}

// Stop останавливает таймеры.
func (s *Service) Stop() error {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closed = true
		for _, t := range s.timers {
			t.Stop()
		}
		s.timers = make(map[string]*time.Timer)
		log.Println("scheduler: module stopped")
	})
	return nil
}
