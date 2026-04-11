package orchestrator

import (
	"context"
	"testing"
	"time"

	"modulr/core/aiengine"
	coreevents "modulr/core/events"
)

func TestPipelineEnrichUsesContextScope(t *testing.T) {
	pipe := NewPipeline(coreevents.NewMemoryBus(), NewModuleRegistry(), NewMonitor(), 0.7)
	evt := coreevents.Event{
		Name:    coreevents.V1MessageReceived,
		Context: map[string]any{"scope": "travel"},
	}

	if err := pipe.Enrich(context.Background(), &evt); err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}
	if evt.Scope != "travel" {
		t.Fatalf("Enrich scope = %q, want travel", evt.Scope)
	}
}

func TestPipelineEnrichRejectsInvalidScope(t *testing.T) {
	pipe := NewPipeline(coreevents.NewMemoryBus(), NewModuleRegistry(), NewMonitor(), 0.7)
	evt := coreevents.Event{
		Name:    coreevents.V1MessageReceived,
		Context: map[string]any{"scope": "unknown"},
	}

	if err := pipe.Enrich(context.Background(), &evt); err == nil {
		t.Fatalf("expected invalid scope error")
	}
}

func TestPipelineDispatchFilterAppliesThresholdScopeAndRegistry(t *testing.T) {
	reg := NewModuleRegistry()
	if err := reg.RegisterModule("finance", []string{"create_transaction"}); err != nil {
		t.Fatalf("RegisterModule returned error: %v", err)
	}
	pipe := NewPipeline(coreevents.NewMemoryBus(), reg, NewMonitor(), 0.8)

	decs := []aiengine.Decision{
		{ID: "ok", Target: "finance", Action: "create_transaction", Confidence: 0.91, Scope: "business"},
		{ID: "low", Target: "finance", Action: "create_transaction", Confidence: 0.79, Scope: "business"},
		{ID: "bad_scope", Target: "finance", Action: "create_transaction", Confidence: 0.95, Scope: "unknown"},
		{ID: "bad_endpoint", Target: "media", Action: "register", Confidence: 0.95, Scope: "business"},
	}

	filtered := pipe.DispatchFilter("business", nil, nil, decs)
	if len(filtered) != 1 || filtered[0].ID != "ok" {
		t.Fatalf("DispatchFilter returned %+v, want only decision ok", filtered)
	}
}

func TestPipelineDispatchFilterRejectsCrossScopeWithoutExplicitPolicy(t *testing.T) {
	reg := NewModuleRegistry()
	if err := reg.RegisterModule("finance", []string{"create_transaction"}); err != nil {
		t.Fatalf("RegisterModule returned error: %v", err)
	}
	pipe := NewPipeline(coreevents.NewMemoryBus(), reg, NewMonitor(), 0.8)

	filtered := pipe.DispatchFilter("personal", nil, nil, []aiengine.Decision{
		{ID: "cross", Target: "finance", Action: "create_transaction", Confidence: 0.95, Scope: "business"},
	})
	if len(filtered) != 0 {
		t.Fatalf("expected cross-scope decision to be filtered out, got %+v", filtered)
	}
}

func TestPipelineDispatchFilterAllowsCrossScopeFromMetadataPolicy(t *testing.T) {
	reg := NewModuleRegistry()
	if err := reg.RegisterModule("finance", []string{"create_transaction"}); err != nil {
		t.Fatalf("RegisterModule returned error: %v", err)
	}
	pipe := NewPipeline(coreevents.NewMemoryBus(), reg, NewMonitor(), 0.8)

	filtered := pipe.DispatchFilter("personal", nil, map[string]any{
		"allowed_scopes": []string{"business"},
	}, []aiengine.Decision{
		{ID: "cross", Target: "finance", Action: "create_transaction", Confidence: 0.95, Scope: "business"},
	})
	if len(filtered) != 1 || filtered[0].ID != "cross" {
		t.Fatalf("expected metadata policy to allow cross-scope decision, got %+v", filtered)
	}
}

func TestPipelineDispatchPublishesSummaryAndTargetEvent(t *testing.T) {
	bus := coreevents.NewMemoryBus()
	reg := NewModuleRegistry()
	if err := reg.RegisterModule("finance", []string{"create_transaction"}); err != nil {
		t.Fatalf("RegisterModule returned error: %v", err)
	}
	mon := NewMonitor()
	pipe := NewPipeline(bus, reg, mon, 0.7)

	eventsCh := make(chan coreevents.Event, 4)
	bus.SubscribeAll(func(ctx context.Context, evt coreevents.Event) {
		eventsCh <- evt
	})

	decision := aiengine.Decision{
		ID:         "dec-1",
		Target:     "finance",
		Action:     "create_transaction",
		Parameters: map[string]any{"amount": 10},
		Confidence: 0.95,
		Scope:      "business",
	}

	if err := pipe.Dispatch(context.Background(), 42, "tr-1", "business", []aiengine.Decision{decision}); err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	got := collectCoreEvents(t, eventsCh, 2)
	seen := make(map[coreevents.Name]coreevents.Event, len(got))
	for _, evt := range got {
		seen[evt.Name] = evt
	}

	if _, ok := seen[coreevents.V1OrchestratorActionDispatch]; !ok {
		t.Fatalf("dispatch summary event was not published")
	}
	targetName := coreevents.Name("v1.finance.create_transaction")
	targetEvt, ok := seen[targetName]
	if !ok {
		t.Fatalf("target event %q was not published", targetName)
	}
	if targetEvt.ChatID != 42 || targetEvt.Scope != "business" {
		t.Fatalf("target event envelope mismatch: chat=%d scope=%q", targetEvt.ChatID, targetEvt.Scope)
	}
	if gotTrace := targetEvt.Context["trace_id"]; gotTrace != "tr-1" {
		t.Fatalf("target event trace = %v, want tr-1", gotTrace)
	}
	if mon.Snapshot().EventCounts[string(targetName)] != 1 {
		t.Fatalf("monitor did not record target event")
	}
}

func collectCoreEvents(t *testing.T, ch <-chan coreevents.Event, n int) []coreevents.Event {
	t.Helper()

	out := make([]coreevents.Event, 0, n)
	deadline := time.After(time.Second)
	for len(out) < n {
		select {
		case evt := <-ch:
			out = append(out, evt)
		case <-deadline:
			t.Fatalf("timed out waiting for %d core events, got %d", n, len(out))
		}
	}
	return out
}
