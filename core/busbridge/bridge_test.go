package busbridge

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	coreevents "modulr/core/events"
	"modulr/events"
)

func TestBridgeMirrorsDomainEventToCoreWithTrace(t *testing.T) {
	domain := events.NewBus(0)
	core := coreevents.NewMemoryBus()
	bridge := New(domain, core)

	if err := bridge.Attach(context.Background()); err != nil {
		t.Fatalf("Attach returned error: %v", err)
	}

	coreCh := make(chan coreevents.Event, 1)
	core.Subscribe(coreevents.Name("v1.message.received"), func(ctx context.Context, evt coreevents.Event) {
		coreCh <- evt
	})

	domain.Publish(events.Event{
		Name:    events.Name("v1.message.received"),
		Payload: map[string]any{"text": "hello"},
		Source:  "telegram",
		TraceID: "tr-domain-1",
		Context: map[string]any{
			"chat_id": int64(77),
			"scope":   "personal",
		},
	})

	select {
	case evt := <-coreCh:
		if evt.ChatID != 77 || evt.Scope != "personal" {
			t.Fatalf("core envelope mismatch: chat=%d scope=%q", evt.ChatID, evt.Scope)
		}
		if evt.Context["trace_id"] != "tr-domain-1" {
			t.Fatalf("core trace_id = %v, want tr-domain-1", evt.Context["trace_id"])
		}
		if evt.Context["source"] != "telegram" {
			t.Fatalf("core source = %v, want telegram", evt.Context["source"])
		}
	case <-time.After(time.Second):
		t.Fatalf("domain event was not mirrored to core")
	}
}

func TestBridgeAppliesCoreAliasAndPreventsLoop(t *testing.T) {
	domain := events.NewBus(0)
	core := coreevents.NewMemoryBus()
	bridge := New(domain, core)
	bridge.RegisterCoreAlias(coreevents.Name("v1.calendar.create_event"), events.V1CalendarCreated, nil)

	if err := bridge.Attach(context.Background()); err != nil {
		t.Fatalf("Attach returned error: %v", err)
	}

	domainCh := make(chan events.Event, 1)
	domain.Subscribe(events.V1CalendarCreated, func(evt events.Event) {
		domainCh <- evt
	})

	var coreSeen atomic.Int32
	core.SubscribeAll(func(ctx context.Context, evt coreevents.Event) {
		coreSeen.Add(1)
	})

	if err := core.Publish(context.Background(), coreevents.Event{
		Name:   coreevents.Name("v1.calendar.create_event"),
		ChatID: 15,
		Scope:  "work",
		Context: map[string]any{
			"trace_id": "tr-core-1",
			"event_id": "evt-1",
		},
		Payload: map[string]any{"title": "Sync"},
	}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case evt := <-domainCh:
		if evt.Name != events.V1CalendarCreated {
			t.Fatalf("domain name = %q, want %q", evt.Name, events.V1CalendarCreated)
		}
		if evt.TraceID != "tr-core-1" {
			t.Fatalf("domain trace = %q, want tr-core-1", evt.TraceID)
		}
		if evt.Context["scope"] != "work" || evt.Context["chat_id"] != int64(15) {
			t.Fatalf("domain context mismatch: %+v", evt.Context)
		}
	case <-time.After(time.Second):
		t.Fatalf("core event was not mirrored to domain")
	}

	time.Sleep(100 * time.Millisecond)
	if coreSeen.Load() != 1 {
		t.Fatalf("expected exactly one core publish, got %d", coreSeen.Load())
	}
}
