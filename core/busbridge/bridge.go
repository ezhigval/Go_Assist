package busbridge

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	coreevents "modulr/core/events"
	"modulr/events"
)

const (
	metaOrigin             = "_busbridge_origin"
	metaOriginalCoreName   = "_busbridge_core_name"
	metaOriginalDomainName = "_busbridge_domain_name"
	originCore             = "core"
	originDomain           = "domain"
)

// PayloadMapper позволяет переименовать/подготовить payload при маршруте core -> domain.
type PayloadMapper func(payload any, evt coreevents.Event) any

type coreAllSubscriber interface {
	SubscribeAll(coreevents.Handler)
}

type coreRoute struct {
	domainName events.Name
	mapPayload PayloadMapper
}

// Bridge связывает domain bus и core bus в одном процессе.
type Bridge struct {
	mu      sync.Mutex
	domain  *events.Bus
	core    coreevents.EventBus
	coreAll coreAllSubscriber

	rootCtx context.Context
	started bool
	routes  map[coreevents.Name]coreRoute
}

// New создаёт адаптер шин. Attach выполняет фактическую подписку.
func New(domain *events.Bus, core coreevents.EventBus) *Bridge {
	b := &Bridge{
		domain: domain,
		core:   core,
		routes: make(map[coreevents.Name]coreRoute),
	}
	if sub, ok := core.(coreAllSubscriber); ok {
		b.coreAll = sub
	}
	return b
}

// RegisterCoreAlias задаёт alias/mapper для core -> domain.
// Если Attach уже выполнен и core bus не умеет SubscribeAll, подписка на новое имя будет добавлена сразу.
func (b *Bridge) RegisterCoreAlias(coreName coreevents.Name, domainName events.Name, mapper PayloadMapper) {
	if coreName == "" || domainName == "" {
		return
	}

	b.mu.Lock()
	b.routes[coreName] = coreRoute{
		domainName: domainName,
		mapPayload: mapper,
	}
	started := b.started
	coreAll := b.coreAll
	b.mu.Unlock()

	if started && coreAll == nil && b.core != nil {
		b.core.Subscribe(coreName, b.onCoreEvent)
	}
}

// Attach подписывает обе шины друг на друга. Повторный вызов безопасен.
func (b *Bridge) Attach(ctx context.Context) error {
	if b.domain == nil || b.core == nil {
		return fmt.Errorf("busbridge: nil domain or core bus")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return nil
	}
	b.rootCtx = ctx
	b.started = true
	b.mu.Unlock()

	b.domain.SubscribeAll(b.onDomainEvent)

	if b.coreAll != nil {
		b.coreAll.SubscribeAll(b.onCoreEvent)
		return nil
	}

	names := map[coreevents.Name]struct{}{
		coreevents.V1MessageReceived:             {},
		coreevents.V1OrchestratorActionDispatch:  {},
		coreevents.V1OrchestratorFallback:        {},
		coreevents.V1OrchestratorDecisionOutcome: {},
		coreevents.V1AIAnalyzeRequest:            {},
		coreevents.V1AIAnalyzeResult:             {},
	}
	for _, name := range b.routeNames() {
		names[name] = struct{}{}
	}
	for name := range names {
		b.core.Subscribe(name, b.onCoreEvent)
	}
	return nil
}

func (b *Bridge) routeNames() []coreevents.Name {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]coreevents.Name, 0, len(b.routes))
	for name := range b.routes {
		out = append(out, name)
	}
	return out
}

func (b *Bridge) onDomainEvent(evt events.Event) {
	if contextMarker(evt.Context) == originCore {
		return
	}

	coreEvt := domainToCore(evt)
	ctx := b.publishContext(coreEvt.Context)
	_ = b.core.Publish(ctx, coreEvt)
}

func (b *Bridge) onCoreEvent(ctx context.Context, evt coreevents.Event) {
	if contextMarker(evt.Context) == originDomain {
		return
	}

	_ = ctx
	b.domain.Publish(b.coreToDomain(evt))
}

func (b *Bridge) publishContext(meta map[string]any) context.Context {
	b.mu.Lock()
	ctx := b.rootCtx
	b.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID := stringFromAny(meta["trace_id"]); traceID != "" {
		return events.WithTraceID(ctx, traceID)
	}
	return ctx
}

func (b *Bridge) coreToDomain(evt coreevents.Event) events.Event {
	ctx := cloneMap(evt.Context)
	ctx[metaOrigin] = originCore
	ctx[metaOriginalCoreName] = string(evt.Name)

	if evt.ChatID != 0 {
		ctx["chat_id"] = evt.ChatID
	}
	if evt.Scope != "" {
		ctx["scope"] = evt.Scope
		if _, ok := ctx["context"]; !ok {
			ctx["context"] = evt.Scope
		}
	}
	if len(evt.Tags) > 0 {
		ctx["tags"] = append([]string(nil), evt.Tags...)
	}

	route := b.lookupRoute(evt.Name)
	payload := evt.Payload
	if route.mapPayload != nil {
		payload = route.mapPayload(payload, evt)
	}

	out := events.Event{
		Name:    route.domainName,
		Payload: payload,
		Source:  sourceFromContext(ctx),
		TraceID: stringFromAny(ctx["trace_id"]),
		Context: ctx,
	}
	if id := stringFromAny(ctx["event_id"]); id != "" {
		out.ID = id
	}
	return out
}

func (b *Bridge) lookupRoute(name coreevents.Name) coreRoute {
	b.mu.Lock()
	defer b.mu.Unlock()
	if route, ok := b.routes[name]; ok {
		return route
	}
	return coreRoute{domainName: events.Name(name)}
}

func domainToCore(evt events.Event) coreevents.Event {
	ctx := cloneMap(evt.Context)
	ctx[metaOrigin] = originDomain
	ctx[metaOriginalDomainName] = string(evt.Name)

	if evt.TraceID != "" {
		ctx["trace_id"] = evt.TraceID
	}
	if evt.Source != "" {
		ctx["source"] = evt.Source
	}
	if evt.ID != "" {
		ctx["event_id"] = evt.ID
	}

	scope := firstNonEmpty(
		segmentString(ctx["scope"]),
		segmentString(ctx["context"]),
		segmentString(ctx["segment"]),
	)
	if scope != "" {
		ctx["scope"] = scope
	}

	return coreevents.Event{
		Name:    coreevents.Name(evt.Name),
		Payload: evt.Payload,
		Context: ctx,
		ChatID:  int64FromAny(ctx["chat_id"]),
		Scope:   scope,
		Tags:    stringSlice(ctx["tags"]),
	}
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return make(map[string]any)
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func contextMarker(meta map[string]any) string {
	return stringFromAny(meta[metaOrigin])
}

func sourceFromContext(meta map[string]any) string {
	if source := stringFromAny(meta["source"]); source != "" {
		return source
	}
	return "core/busbridge"
}

func segmentString(v any) string {
	return string(events.ParseSegmentFromAny(v))
}

func stringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s := stringFromAny(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
