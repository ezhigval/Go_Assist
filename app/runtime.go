package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"modulr/core/aiengine"
	"modulr/core/busbridge"
	coreevents "modulr/core/events"
	"modulr/core/orchestrator"
	"modulr/events"
	"modulr/finance"
	"modulr/knowledge"
	"modulr/metrics"
	"modulr/scheduler"
	"modulr/tracker"
)

// InboundMessage нормализованное входящее сообщение транспорта.
type InboundMessage struct {
	ChatID   int64
	UserID   int64
	Username string
	Text     string
	Scope    string
	Source   string
	Tags     []string
	Context  map[string]any
}

// HandleResult итог обработки входящего сообщения для transport-слоя.
type HandleResult struct {
	TraceID     string
	Status      string
	Scope       string
	ActionEvent string
	DecisionID  string
	ModelID     string
	Reason      string
	Error       string
}

// Runtime собирает in-memory MVP сценарий message -> orchestrator -> domain bus.
type Runtime struct {
	mu sync.Mutex

	domainBus *events.Bus
	coreBus   *coreevents.MemoryBus
	bridge    *busbridge.Bridge
	ai        aiengine.AIEngine
	orch      *orchestrator.Orchestrator
	store     *events.MemoryStorage

	tracker   *tracker.Service
	finance   *finance.Service
	knowledge *knowledge.Service
	metrics   *metrics.Service
	scheduler *scheduler.Service
	journal   EventJournal

	waitMu  sync.Mutex
	waiters map[string][]chan HandleResult

	started bool
}

// NewRuntime создаёт локальную сборку для v0.3.
func NewRuntime(opts ...RuntimeOption) *Runtime {
	domainBus := events.NewBus(128)
	coreBus := coreevents.NewMemoryBus()
	bridge := busbridge.New(domainBus, coreBus)
	bridge.RegisterCoreAlias(coreevents.Name("v1.calendar.create_event"), events.V1CalendarCreated, mapCalendarPayload)

	store := events.NewMemoryStorage()
	rt := &Runtime{
		domainBus: domainBus,
		coreBus:   coreBus,
		bridge:    bridge,
		ai:        aiengine.NewEngine(),
		store:     store,
		tracker:   tracker.NewService(store, domainBus),
		finance:   finance.NewService(store, domainBus),
		knowledge: knowledge.NewService(store, domainBus),
		metrics:   metrics.NewService(metrics.LoadConfig(), domainBus, events.NewMemoryIdempotency()),
		scheduler: scheduler.NewService(scheduler.LoadConfig(), domainBus, events.NewMemoryIdempotency()),
		waiters:   make(map[string][]chan HandleResult),
	}
	rt.orch = orchestrator.NewOrchestrator(coreBus, rt.ai, 0.7)
	for _, opt := range opts {
		if opt != nil {
			opt(rt)
		}
	}
	return rt
}

// Start связывает шины, подписки модулей и оркестратор.
func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	if err := r.bridge.Attach(ctx); err != nil {
		return err
	}
	r.domainBus.Subscribe(EventOrchestratorOutcome, r.onOutcome)
	r.domainBus.Subscribe(EventOrchestratorFallback, r.onFallback)

	r.tracker.RegisterSubscriptions(r.domainBus)
	r.finance.RegisterSubscriptions(r.domainBus)
	r.knowledge.RegisterSubscriptions(r.domainBus)
	registerActionHandlers(actionHandlerConfig{
		bus:       r.domainBus,
		tracker:   r.tracker,
		finance:   r.finance,
		knowledge: r.knowledge,
	})
	if err := r.metrics.Start(ctx); err != nil {
		return err
	}

	if err := r.scheduler.Start(ctx); err != nil {
		return err
	}
	if err := r.orch.Start(ctx); err != nil {
		return err
	}
	r.started = true
	return nil
}

// Stop останавливает runtime.
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	started := r.started
	r.started = false
	r.mu.Unlock()
	if !started {
		return nil
	}

	if err := r.orch.Stop(ctx); err != nil {
		return err
	}
	if err := r.scheduler.Stop(); err != nil {
		return err
	}
	return r.metrics.Stop()
}

// HandleMessage публикует входящее сообщение на шину ядра.
func (r *Runtime) HandleMessage(ctx context.Context, msg InboundMessage) (string, error) {
	if strings.TrimSpace(msg.Text) == "" {
		return "", fmt.Errorf("app/runtime: empty text")
	}
	scope := msg.Scope
	if scope == "" {
		scope = string(events.DefaultSegment())
	}
	if !events.IsValidSegment(events.Segment(scope)) {
		return "", fmt.Errorf("app/runtime: invalid scope %q", scope)
	}

	traceID := events.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	meta := cloneContext(msg.Context)
	meta["trace_id"] = traceID
	meta["source"] = firstNonEmptyString(msg.Source, "telegram")
	meta["user_id"] = msg.UserID
	meta["username"] = msg.Username

	err := r.coreBus.Publish(events.WithTraceID(ctx, traceID), coreevents.Event{
		Name:    coreevents.V1MessageReceived,
		Payload: map[string]any{"text": msg.Text},
		ChatID:  msg.ChatID,
		Scope:   scope,
		Tags:    append([]string(nil), msg.Tags...),
		Context: meta,
	})
	if err != nil {
		return "", err
	}
	r.writeJournal(ctx, JournalRecord{
		TraceID:   traceID,
		ChatID:    msg.ChatID,
		Scope:     scope,
		EventName: string(coreevents.V1MessageReceived),
		Status:    "accepted",
		Source:    firstNonEmptyString(msg.Source, "telegram"),
		Payload: map[string]any{
			"text": msg.Text,
			"tags": append([]string(nil), msg.Tags...),
		},
		Metadata: map[string]any{
			"user_id":  msg.UserID,
			"username": msg.Username,
		},
		CreatedAt: time.Now(),
	})
	return traceID, nil
}

// HandleMessageSync публикует сообщение и ждёт outcome/fallback для transport-ответа.
func (r *Runtime) HandleMessageSync(ctx context.Context, msg InboundMessage) (HandleResult, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
	}

	traceID := events.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
		ctx = events.WithTraceID(ctx, traceID)
	}

	ch := make(chan HandleResult, 1)
	r.addWaiter(traceID, ch)
	defer r.removeWaiter(traceID, ch)

	if _, err := r.HandleMessage(ctx, msg); err != nil {
		return HandleResult{}, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		result := HandleResult{
			TraceID: traceID,
			Status:  "timeout",
			Scope:   firstNonEmptyString(msg.Scope, string(events.DefaultSegment())),
			Error:   ctx.Err().Error(),
		}
		r.writeJournal(context.Background(), JournalRecord{
			TraceID:   traceID,
			ChatID:    msg.ChatID,
			Scope:     result.Scope,
			EventName: string(EventTransportResponseTimeout),
			Status:    "timeout",
			Source:    "app/runtime",
			Payload: map[string]any{
				"error": ctx.Err().Error(),
			},
			Metadata: map[string]any{
				"source": firstNonEmptyString(msg.Source, "telegram"),
			},
			CreatedAt: time.Now(),
		})
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, nil
		}
		return result, ctx.Err()
	}
}

// Store возвращает in-memory storage для тестов и отладки.
func (r *Runtime) Store() *events.MemoryStorage {
	return r.store
}

// Orchestrator возвращает оркестратор для метрик/диагностики.
func (r *Runtime) Orchestrator() *orchestrator.Orchestrator {
	return r.orch
}

// Metrics возвращает runtime metrics-сервис для observability/диагностики.
func (r *Runtime) Metrics() *metrics.Service {
	return r.metrics
}

func (r *Runtime) onOutcome(evt events.Event) {
	payload, _ := payloadToMap(evt.Payload)
	traceID := firstNonEmptyString(evt.TraceID, stringFromAny(evt.Context["trace_id"]))
	if traceID == "" {
		return
	}
	result := HandleResult{
		TraceID:     traceID,
		Status:      statusFromOutcome(payload),
		Scope:       firstNonEmptyString(stringFromAny(payload["scope"]), stringFromAny(evt.Context["scope"])),
		ActionEvent: stringFromAny(payload["action_event"]),
		DecisionID:  stringFromAny(payload["decision_id"]),
		ModelID:     stringFromAny(payload["model_id"]),
		Error:       stringFromAny(payload["error"]),
	}
	r.writeJournal(context.Background(), JournalRecord{
		TraceID:   traceID,
		ChatID:    int64FromAny(evt.Context["chat_id"]),
		Scope:     firstNonEmptyString(result.Scope, string(events.DefaultSegment())),
		EventName: string(EventOrchestratorOutcome),
		Status:    result.Status,
		Source:    evt.Source,
		Payload:   cloneContext(payload),
		Metadata:  cloneContext(evt.Context),
		CreatedAt: time.Now(),
	})
	r.notifyWaiters(traceID, result)
}

func (r *Runtime) onFallback(evt events.Event) {
	payload, _ := payloadToMap(evt.Payload)
	traceID := firstNonEmptyString(evt.TraceID, stringFromAny(evt.Context["trace_id"]))
	if traceID == "" {
		return
	}
	result := HandleResult{
		TraceID: traceID,
		Status:  "fallback",
		Scope:   firstNonEmptyString(stringFromAny(evt.Context["scope"]), stringFromAny(payload["scope"])),
		Reason:  stringFromAny(payload["reason"]),
	}
	r.writeJournal(context.Background(), JournalRecord{
		TraceID:   traceID,
		ChatID:    int64FromAny(evt.Context["chat_id"]),
		Scope:     firstNonEmptyString(result.Scope, string(events.DefaultSegment())),
		EventName: string(EventOrchestratorFallback),
		Status:    "fallback",
		Source:    evt.Source,
		Payload:   cloneContext(payload),
		Metadata:  cloneContext(evt.Context),
		CreatedAt: time.Now(),
	})
	r.notifyWaiters(traceID, result)
}

func (r *Runtime) addWaiter(traceID string, ch chan HandleResult) {
	r.waitMu.Lock()
	defer r.waitMu.Unlock()
	r.waiters[traceID] = append(r.waiters[traceID], ch)
}

func (r *Runtime) removeWaiter(traceID string, ch chan HandleResult) {
	r.waitMu.Lock()
	defer r.waitMu.Unlock()

	list := r.waiters[traceID]
	for i := range list {
		if list[i] == ch {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(list) == 0 {
		delete(r.waiters, traceID)
		return
	}
	r.waiters[traceID] = list
}

func (r *Runtime) notifyWaiters(traceID string, result HandleResult) {
	r.waitMu.Lock()
	list := append([]chan HandleResult(nil), r.waiters[traceID]...)
	delete(r.waiters, traceID)
	r.waitMu.Unlock()

	for _, ch := range list {
		select {
		case ch <- result:
		default:
		}
	}
}

func mapCalendarPayload(payload any, evt coreevents.Event) any {
	m, ok := payloadToMap(payload)
	if !ok {
		return map[string]any{
			"title":   fmt.Sprint(payload),
			"context": evt.Scope,
		}
	}
	if _, ok := m["context"]; !ok {
		m["context"] = evt.Scope
	}
	if _, ok := m["start"]; !ok {
		m["start"] = time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	}
	return m
}

func cloneContext(src map[string]any) map[string]any {
	if len(src) == 0 {
		return make(map[string]any)
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func statusFromOutcome(payload map[string]any) string {
	switch value := payload["ok"].(type) {
	case bool:
		if value {
			return "completed"
		}
	case string:
		if value == "true" {
			return "completed"
		}
	}
	return "failed"
}

func (r *Runtime) writeJournal(ctx context.Context, record JournalRecord) {
	if r == nil || r.journal == nil {
		return
	}
	if record.TraceID == "" || record.EventName == "" {
		return
	}
	if record.Scope == "" {
		record.Scope = string(events.DefaultSegment())
	}
	if record.Source == "" {
		record.Source = "app/runtime"
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.Payload = cloneContext(record.Payload)
	record.Metadata = cloneContext(record.Metadata)

	if err := r.journal.WriteEvent(ctx, record); err != nil {
		log.Printf("app/runtime: journal write [%s]: %v", record.EventName, err)
	}
}
