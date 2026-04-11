package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"modulr/core/aiengine"
	coreevents "modulr/core/events"
)

var _ OrchestratorAPI = (*Orchestrator)(nil)

// Orchestrator центральный мозг: маршрутизация, валидация решений AI, мониторинг.
type Orchestrator struct {
	mu        sync.RWMutex
	bus       coreevents.EventBus
	ai        aiengine.AIEngine
	reg       *ModuleRegistry
	mon       *Monitor
	pipe      *Pipeline
	threshold float64

	cancel  context.CancelFunc
	rootCtx context.Context
	wg      sync.WaitGroup
	started bool
}

// NewOrchestrator собирает оркестратор с дефолтным реестром модулей.
func NewOrchestrator(bus coreevents.EventBus, ai aiengine.AIEngine, threshold float64) *Orchestrator {
	reg := NewModuleRegistry()
	mon := NewMonitor()
	if threshold <= 0 {
		threshold = 0.7
	}
	o := &Orchestrator{
		bus:       bus,
		ai:        ai,
		reg:       reg,
		mon:       mon,
		pipe:      NewPipeline(bus, reg, mon, threshold),
		threshold: threshold,
	}
	o.seedDefaultModules()
	return o
}

func (o *Orchestrator) seedDefaultModules() {
	// Операции LEGO (см. README / ARCHITECTURE): реестр для валидации цепочек ИИ.
	_ = o.reg.RegisterModule("calendar", []string{"create_event", "find_free_slot", "resolve_conflict"})
	_ = o.reg.RegisterModule("tracker", []string{"create_reminder", "create_task", "log_progress"})
	_ = o.reg.RegisterModule("maps", []string{"build_route", "set_geofence"})
	_ = o.reg.RegisterModule("knowledge", []string{"save_query", "save_note", "link_entities"})
	_ = o.reg.RegisterModule("finance", []string{"create_transaction", "set_budget", "forecast_cashflow"})
	_ = o.reg.RegisterModule("contacts", []string{"add_contact", "log_touch", "find_decision_maker"})
	_ = o.reg.RegisterModule("metrics", []string{"log_metric", "calculate_trend", "detect_anomaly"})
	_ = o.reg.RegisterModule("logistics", []string{"build_route", "track_shipment", "optimize_path"})
	_ = o.reg.RegisterModule("inventory", []string{"register_item", "track_condition", "schedule_maintenance"})
	_ = o.reg.RegisterModule("notifications", []string{"schedule_alert", "route_by_channel", "escalate"})
	_ = o.reg.RegisterModule("media", []string{"register", "tag", "link"})
}

// RegisterModule делегирует в реестр.
func (o *Orchestrator) RegisterModule(ctx context.Context, name string, actions []string) error {
	_ = ctx
	return o.reg.RegisterModule(name, actions)
}

// Start подписывается на ключевые события шины.
func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.started {
		return nil
	}
	if o.bus == nil || o.ai == nil {
		return fmt.Errorf("orchestrator: nil bus or ai")
	}
	o.rootCtx, o.cancel = context.WithCancel(ctx)
	o.started = true

	o.bus.Subscribe(coreevents.V1MessageReceived, o.onMessageReceived)
	o.bus.Subscribe(coreevents.V1OrchestratorDecisionOutcome, o.onDecisionOutcome)

	if err := o.ai.Start(o.rootCtx); err != nil {
		return err
	}
	log.Println("orchestrator: started")
	return nil
}

// Stop отменяет контекст и ждёт завершения активных ProcessEvent.
func (o *Orchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	if o.cancel != nil {
		o.cancel()
	}
	o.started = false
	o.mu.Unlock()

	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	_ = o.ai.Stop(context.Background())
	log.Println("orchestrator: stopped")
	return nil
}

func (o *Orchestrator) onMessageReceived(ctx context.Context, e coreevents.Event) {
	if err := o.ProcessEvent(ctx, e); err != nil {
		log.Printf("orchestrator: process message: %v", err)
		o.mon.RecordError()
	}
}

func (o *Orchestrator) onDecisionOutcome(ctx context.Context, e coreevents.Event) {
	o.wg.Add(1)
	defer o.wg.Done()

	start := time.Now()
	defer StepTimer(o.mon, "feedback", start)

	m, ok := e.Payload.(map[string]any)
	if !ok {
		o.mon.RecordError()
		return
	}
	fb := aiengine.Feedback{
		ModelID:    fmt.Sprint(m["model_id"]),
		DecisionID: fmt.Sprint(m["decision_id"]),
		Error:      fmt.Sprint(m["error"]),
		Scope:      firstString(e.Scope, fmt.Sprint(m["scope"])),
	}
	switch v := m["ok"].(type) {
	case bool:
		fb.OK = v
	case string:
		fb.OK = v == "true"
	}
	switch v := m["latency_ms"].(type) {
	case int64:
		fb.LatencyMs = v
	case float64:
		fb.LatencyMs = int64(v)
	case int:
		fb.LatencyMs = int64(v)
	}
	if err := o.ai.Feedback(ctx, fb); err != nil {
		log.Printf("orchestrator: ai feedback: %v", err)
		o.mon.RecordError()
	}
}

func firstString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ProcessEvent выполняет конвейер для события.
func (o *Orchestrator) ProcessEvent(ctx context.Context, e coreevents.Event) error {
	o.wg.Add(1)
	defer o.wg.Done()

	tAll := time.Now()
	defer StepTimer(o.mon, "process_total", tAll)

	ev := e
	if ev.Context == nil {
		ev.Context = make(map[string]any)
	}

	t := time.Now()
	if err := o.pipe.Enrich(ctx, &ev); err != nil {
		o.mon.RecordError()
		return err
	}
	StepTimer(o.mon, "enrich", t)

	t = time.Now()
	if err := o.pipe.Validate(ctx, &ev); err != nil {
		o.mon.RecordError()
		return err
	}
	StepTimer(o.mon, "validate", t)

	o.mon.RecordEvent(ev.Name)
	o.mon.AppendChatHistory(ev.ChatID, HistoryItem{
		Time:    time.Now(),
		Name:    string(ev.Name),
		Scope:   ev.Scope,
		Summary: ExtractText(ev.Payload),
	})

	if ev.Name != coreevents.V1MessageReceived {
		return nil
	}

	analyzeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	t = time.Now()
	req := BuildAIRequest(ev, ExtractText(ev.Payload))
	// Прокидываем последние реплики в метаданные для будущего LLM
	req.Metadata["history"] = o.mon.ChatHistory(ev.ChatID)

	decs, err := o.ai.Analyze(analyzeCtx, req)
	StepTimer(o.mon, "ai_analyze", t)

	if err != nil {
		o.handleFallback(ctx, ev, fmt.Errorf("ai analyze: %w", err))
		return nil
	}
	if len(decs) == 0 {
		o.handleFallback(ctx, ev, fmt.Errorf("ai analyze: empty decisions"))
		return nil
	}

	decs = o.pipe.Prioritize(decs)
	filtered := o.pipe.DispatchFilter(ev.Scope, ev.Tags, ev.Context, decs)
	if len(filtered) == 0 {
		o.handleFallback(ctx, ev, fmt.Errorf("orchestrator: no decisions passed filters"))
		return nil
	}

	t = time.Now()
	trace := fmt.Sprint(ev.Context["trace_id"])
	if trace == "" {
		trace = fmt.Sprintf("tr_%d", time.Now().UnixNano())
		ev.Context["trace_id"] = trace
	}
	if err := o.pipe.Dispatch(ctx, ev.ChatID, trace, ev.Scope, filtered); err != nil {
		o.mon.RecordError()
		o.handleFallback(ctx, ev, err)
		return nil
	}
	StepTimer(o.mon, "dispatch", t)

	return nil
}

func (o *Orchestrator) handleFallback(ctx context.Context, e coreevents.Event, cause error) {
	o.mon.RecordError()
	if o.bus == nil {
		return
	}
	reason := "unknown"
	if cause != nil {
		reason = cause.Error()
	}
	payload := map[string]any{
		"original":   e,
		"reason":     reason,
		"suggestion": "manual_confirm_or_rules",
	}
	fbEvt := coreevents.Event{
		Name:    coreevents.V1OrchestratorFallback,
		Payload: payload,
		ChatID:  e.ChatID,
		Scope:   e.Scope,
		Tags:    []string{"fallback", "orchestrator"},
		Context: map[string]any{"trace_id": e.Context["trace_id"]},
	}
	// STUB: Fallback escalation requires v1.orchestrator.fallback.requested handlers (operator, config rules) with trace_id logging.
	_ = o.bus.Publish(ctx, fbEvt)
}

// GetStats возвращает снимок монитора.
func (o *Orchestrator) GetStats(ctx context.Context) (Stats, error) {
	_ = ctx
	return o.mon.Snapshot(), nil
}
