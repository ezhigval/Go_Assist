package metrics

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"modulr/events"
)

var _ MetricsAPI = (*Service)(nil)

const maxTraceSummaries = 128

// Service пассивный слушатель событий для счётчиков и отчётности (аномалии/тренды — поверх тех же данных).
type Service struct {
	cfg         Config
	bus         *events.Bus
	idem        events.IdempotencyStore
	mu          sync.Mutex
	counts      map[string]int64
	scopeCounts map[string]int64
	traces      map[string]TraceSummary
	started     bool
}

// NewService создаёт metrics-сервис.
func NewService(cfg Config, bus *events.Bus, idem events.IdempotencyStore) *Service {
	return &Service{
		cfg:         cfg,
		bus:         bus,
		idem:        idem,
		counts:      make(map[string]int64),
		scopeCounts: make(map[string]int64),
		traces:      make(map[string]TraceSummary),
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
	if scope := scopeFromEvent(evt); scope != "" {
		s.scopeCounts[scope]++
	}
	if traceID := traceIDFromEvent(evt); traceID != "" {
		item := s.traces[traceID]
		item.TraceID = traceID
		if scope := scopeFromEvent(evt); scope != "" {
			item.Scope = scope
		}
		item.EventCount++
		item.LastEvent = string(evt.Name)
		item.LastSource = evt.Source
		item.UpdatedAt = evt.Timestamp
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = time.Now()
		}
		s.traces[traceID] = item
		if len(s.traces) > maxTraceSummaries {
			s.evictOldestTrace()
		}
	}
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

// ScopeCounts возвращает агрегаты по scope.
func (s *Service) ScopeCounts() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64, len(s.scopeCounts))
	for k, v := range s.scopeCounts {
		out[k] = v
	}
	return out
}

// Snapshot возвращает события, scope-агрегаты и последние trace-summary.
func (s *Service) Snapshot(limit int) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}

	counts := make(map[string]int64, len(s.counts))
	for k, v := range s.counts {
		counts[k] = v
	}

	scopeCounts := make(map[string]int64, len(s.scopeCounts))
	for k, v := range s.scopeCounts {
		scopeCounts[k] = v
	}

	traces := make([]TraceSummary, 0, len(s.traces))
	for _, trace := range s.traces {
		traces = append(traces, trace)
	}
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].UpdatedAt.After(traces[j].UpdatedAt)
	})
	if len(traces) > limit {
		traces = traces[:limit]
	}

	return Snapshot{
		Counts:      counts,
		ScopeCounts: scopeCounts,
		Traces:      traces,
	}
}

// Stop останавливает модуль.
func (s *Service) Stop() error {
	log.Println("metrics: module stopped")
	return nil
}

func (s *Service) evictOldestTrace() {
	var (
		oldestID string
		oldestAt time.Time
	)
	for traceID, item := range s.traces {
		if oldestID == "" || item.UpdatedAt.Before(oldestAt) {
			oldestID = traceID
			oldestAt = item.UpdatedAt
		}
	}
	if oldestID != "" {
		delete(s.traces, oldestID)
	}
}

func traceIDFromEvent(evt events.Event) string {
	if evt.TraceID != "" {
		return evt.TraceID
	}
	if evt.Context == nil {
		return ""
	}
	switch v := evt.Context["trace_id"].(type) {
	case string:
		return v
	default:
		return ""
	}
}

func scopeFromEvent(evt events.Event) string {
	if evt.Context != nil {
		if scope := string(events.ParseSegmentFromAny(evt.Context["scope"])); scope != "" {
			return scope
		}
		if scope := string(events.ParseSegmentFromAny(evt.Context["context"])); scope != "" {
			return scope
		}
		if scope := string(events.ParseSegmentFromAny(evt.Context["segment"])); scope != "" {
			return scope
		}
	}
	if payload, ok := evt.Payload.(map[string]any); ok {
		if scope := string(events.ParseSegmentFromAny(payload["scope"])); scope != "" {
			return scope
		}
		if scope := string(events.ParseSegmentFromAny(payload["context"])); scope != "" {
			return scope
		}
	}
	return ""
}
