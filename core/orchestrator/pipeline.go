package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"modulr/core/aiengine"
	coreevents "modulr/core/events"
	"modulr/events"
)

// Pipeline конвейер enrich → validate → prioritize → dispatch → (фидбек снаружи по шине).
type Pipeline struct {
	bus       coreevents.EventBus
	reg       *ModuleRegistry
	mon       *Monitor
	threshold float64
}

type DispatchFilterReport struct {
	Allowed  []aiengine.Decision
	Rejected []RejectedDecision
}

type RejectedDecision struct {
	DecisionID string
	Reason     string
}

// NewPipeline создаёт конвейер.
func NewPipeline(bus coreevents.EventBus, reg *ModuleRegistry, mon *Monitor, threshold float64) *Pipeline {
	if threshold <= 0 {
		threshold = 0.7
	}
	return &Pipeline{bus: bus, reg: reg, mon: mon, threshold: threshold}
}

// Enrich нормализует Scope/Tags и дополняет контекст.
func (p *Pipeline) Enrich(ctx context.Context, e *coreevents.Event) error {
	_ = ctx
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	if e.Scope == "" {
		if s, ok := e.Context["scope"].(string); ok && s != "" {
			e.Scope = s
		} else {
			e.Scope = "personal"
		}
	}
	if !events.IsValidSegment(events.Segment(e.Scope)) {
		return fmt.Errorf("pipeline: invalid scope %q", e.Scope)
	}
	e.Context["scope"] = e.Scope
	return nil
}

// Validate проверяет базовую целостность события.
func (p *Pipeline) Validate(ctx context.Context, e *coreevents.Event) error {
	_ = ctx
	if e.Name == "" {
		return fmt.Errorf("pipeline: empty event name")
	}
	return nil
}

// Prioritize сортирует решения по убыванию confidence.
func (p *Pipeline) Prioritize(decs []aiengine.Decision) []aiengine.Decision {
	out := append([]aiengine.Decision(nil), decs...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Confidence > out[j].Confidence
	})
	return out
}

// DispatchFilter возвращает решения, прошедшие порог, реестр эндпоинтов и scope-policy.
func (p *Pipeline) DispatchFilter(scope string, tags []string, metadata map[string]any, decs []aiengine.Decision) []aiengine.Decision {
	return p.DispatchFilterReport(scope, tags, metadata, decs).Allowed
}

// DispatchFilterReport возвращает прошедшие решения и причины отклонения остальных.
func (p *Pipeline) DispatchFilterReport(scope string, tags []string, metadata map[string]any, decs []aiengine.Decision) DispatchFilterReport {
	var report DispatchFilterReport
	for _, d := range decs {
		filtered, rejectReason, allowed := p.filterDecision(scope, tags, metadata, d)
		if allowed {
			report.Allowed = append(report.Allowed, filtered)
			continue
		}
		report.Rejected = append(report.Rejected, RejectedDecision{
			DecisionID: d.ID,
			Reason:     rejectReason,
		})
	}
	return report
}

func (r DispatchFilterReport) Summary() string {
	if len(r.Rejected) == 0 {
		return ""
	}
	counts := make(map[string]int, len(r.Rejected))
	for _, item := range r.Rejected {
		counts[item.Reason]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func (p *Pipeline) filterDecision(scope string, tags []string, metadata map[string]any, d aiengine.Decision) (aiengine.Decision, string, bool) {
	if d.Confidence < p.threshold {
		return d, "below_threshold", false
	}

	effectiveScope := firstNonEmpty(d.Scope, scope)
	if effectiveScope != "" && !events.IsValidSegment(events.Segment(effectiveScope)) {
		return d, "invalid_scope", false
	}
	if scope != "" && !events.ScopeAllowed(events.Segment(scope), events.Segment(effectiveScope), tags, metadata) {
		return d, "scope_denied", false
	}
	if !p.reg.HasEndpoint(d.Target, d.Action) {
		return d, "endpoint_unknown", false
	}

	targetEvent := fmt.Sprintf("v1.%s.%s", d.Target, d.Action)
	roles := events.NormalizeRoles(metadata["roles"])
	if len(roles) != 0 && !events.RolesAllowEvent(roles, targetEvent) {
		return d, "role_denied", false
	}
	if events.MetadataAuthRequired(metadata) && len(roles) == 0 {
		return d, "auth_required", false
	}

	if d.Scope == "" {
		d.Scope = effectiveScope
	}
	return d, "", true
}

// Dispatch публикует сводное событие диспетча и целевые v1.{module}.{action}.
func (p *Pipeline) Dispatch(ctx context.Context, chatID int64, traceID string, scope string, decs []aiengine.Decision) error {
	if p.bus == nil {
		return fmt.Errorf("pipeline: nil bus")
	}
	env := coreevents.Event{
		Name:    coreevents.V1OrchestratorActionDispatch,
		Payload: map[string]any{"decisions": decs, "trace_id": traceID},
		ChatID:  chatID,
		Scope:   scope,
		Tags:    []string{"orchestrator", "dispatch"},
		Context: map[string]any{"trace_id": traceID},
	}
	if err := p.bus.Publish(ctx, env); err != nil {
		return err
	}
	for _, d := range decs {
		name := coreevents.Name(fmt.Sprintf("v1.%s.%s", d.Target, d.Action))
		evt := coreevents.Event{
			Name:    name,
			Payload: d.Parameters,
			ChatID:  chatID,
			Scope:   firstNonEmpty(d.Scope, scope),
			Tags:    []string{"orchestrator", d.Target, d.Action},
			Context: map[string]any{
				"trace_id":    traceID,
				"decision_id": d.ID,
				"model_id":    d.ModelID,
				"confidence":  d.Confidence,
			},
		}
		if err := p.bus.Publish(ctx, evt); err != nil {
			return err
		}
		p.mon.TouchModule(d.Target)
		p.mon.RecordEvent(name)
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// BuildAIRequest собирает запрос к AIEngine из доменного события.
func BuildAIRequest(e coreevents.Event, text string) aiengine.Request {
	return aiengine.Request{
		TraceID:  fmt.Sprint(e.Context["trace_id"]),
		ChatID:   e.ChatID,
		Scope:    e.Scope,
		Text:     text,
		Tags:     append([]string(nil), e.Tags...),
		KindHint: fmt.Sprint(e.Context["kind_hint"]),
		Metadata: e.Context,
	}
}

// ExtractText извлекает текст пользователя из payload (map или строка).
func ExtractText(payload any) string {
	switch v := payload.(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["text"].(string); ok {
			return s
		}
		if s, ok := v["message"].(string); ok {
			return s
		}
		if s, ok := v["body"].(string); ok {
			return s
		}
	}
	return ""
}

// StepTimer обёртка для измерения шага конвейера.
func StepTimer(mon *Monitor, step string, start time.Time) {
	mon.RecordLatency(step, time.Since(start))
}
