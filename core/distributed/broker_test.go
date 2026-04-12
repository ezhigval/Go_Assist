package distributed

import (
	"context"
	"errors"
	"sync"
	"testing"

	coreevents "modulr/core/events"
)

type deliveryCountKey struct {
	group    string
	consumer string
}

func TestMemoryBrokerFanoutAcrossGroupsAndRoundRobinWithinGroup(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	var (
		mu     sync.Mutex
		counts = map[deliveryCountKey]int{}
	)

	record := func(group, consumer string) Handler {
		return func(_ context.Context, delivery Delivery) error {
			mu.Lock()
			defer mu.Unlock()
			counts[deliveryCountKey{group: group, consumer: consumer}]++
			if delivery.Group != group {
				t.Fatalf("delivery group = %q, want %q", delivery.Group, group)
			}
			if delivery.Consumer != consumer {
				t.Fatalf("delivery consumer = %q, want %q", delivery.Consumer, consumer)
			}
			return nil
		}
	}

	if _, err := broker.SubscribeGroup(ctx, "runtime.events", "workers", "worker-a", record("workers", "worker-a")); err != nil {
		t.Fatalf("SubscribeGroup(worker-a) returned error: %v", err)
	}
	if _, err := broker.SubscribeGroup(ctx, "runtime.events", "workers", "worker-b", record("workers", "worker-b")); err != nil {
		t.Fatalf("SubscribeGroup(worker-b) returned error: %v", err)
	}
	if _, err := broker.SubscribeGroup(ctx, "runtime.events", "audit", "audit-1", record("audit", "audit-1")); err != nil {
		t.Fatalf("SubscribeGroup(audit-1) returned error: %v", err)
	}

	for i := 0; i < 4; i++ {
		if err := broker.Publish(ctx, "runtime.events", Envelope{Name: "v1.orchestrator.action.dispatch"}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
	}

	stats := broker.Stats("runtime.events")
	if stats.Published != 4 {
		t.Fatalf("Published = %d, want 4", stats.Published)
	}
	if stats.GroupCount != 2 {
		t.Fatalf("GroupCount = %d, want 2", stats.GroupCount)
	}
	if stats.SubscriberCount != 3 {
		t.Fatalf("SubscriberCount = %d, want 3", stats.SubscriberCount)
	}
	if counts[deliveryCountKey{group: "workers", consumer: "worker-a"}] != 2 {
		t.Fatalf("worker-a deliveries = %d, want 2", counts[deliveryCountKey{group: "workers", consumer: "worker-a"}])
	}
	if counts[deliveryCountKey{group: "workers", consumer: "worker-b"}] != 2 {
		t.Fatalf("worker-b deliveries = %d, want 2", counts[deliveryCountKey{group: "workers", consumer: "worker-b"}])
	}
	if counts[deliveryCountKey{group: "audit", consumer: "audit-1"}] != 4 {
		t.Fatalf("audit-1 deliveries = %d, want 4", counts[deliveryCountKey{group: "audit", consumer: "audit-1"}])
	}
}

func TestMemoryBrokerRecordsHandlerFailuresWithoutBreakingPublish(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	if _, err := broker.SubscribeGroup(ctx, "runtime.events", "workers", "broken-worker", func(_ context.Context, _ Delivery) error {
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("SubscribeGroup returned error: %v", err)
	}

	if err := broker.Publish(ctx, "runtime.events", Envelope{Name: "v1.test"}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	stats := broker.Stats("runtime.events")
	if len(stats.Groups) != 1 {
		t.Fatalf("expected one group in stats, got %+v", stats.Groups)
	}
	if stats.Groups[0].Delivered != 1 || stats.Groups[0].Failed != 1 {
		t.Fatalf("unexpected group stats: %+v", stats.Groups[0])
	}
}

func TestMemoryBrokerLoadScenarioBalancesHighVolumeAcrossGroups(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	const publishes = 4096

	var (
		mu     sync.Mutex
		counts = map[deliveryCountKey]int{}
	)

	subscribeGroup := func(group string, consumers ...string) {
		t.Helper()
		for _, consumer := range consumers {
			consumer := consumer
			if _, err := broker.SubscribeGroup(ctx, "runtime.events", group, consumer, func(_ context.Context, delivery Delivery) error {
				mu.Lock()
				counts[deliveryCountKey{group: group, consumer: consumer}]++
				mu.Unlock()

				if delivery.Topic != "runtime.events" {
					t.Fatalf("delivery topic = %q, want runtime.events", delivery.Topic)
				}
				return nil
			}); err != nil {
				t.Fatalf("SubscribeGroup(%s/%s) returned error: %v", group, consumer, err)
			}
		}
	}

	subscribeGroup("workers", "worker-a", "worker-b", "worker-c", "worker-d")
	subscribeGroup("audit", "audit-a", "audit-b")
	subscribeGroup("metrics", "metrics-a")

	for i := 0; i < publishes; i++ {
		if err := broker.Publish(ctx, "runtime.events", Envelope{
			Name:  "v1.orchestrator.action.dispatch",
			Scope: "business",
		}); err != nil {
			t.Fatalf("Publish(%d) returned error: %v", i, err)
		}
	}

	stats := broker.Stats("runtime.events")
	if stats.Published != publishes {
		t.Fatalf("Published = %d, want %d", stats.Published, publishes)
	}
	if stats.GroupCount != 3 {
		t.Fatalf("GroupCount = %d, want 3", stats.GroupCount)
	}
	if stats.SubscriberCount != 7 {
		t.Fatalf("SubscriberCount = %d, want 7", stats.SubscriberCount)
	}

	assertBalancedCounts(t, counts, "workers", publishes/4, "worker-a", "worker-b", "worker-c", "worker-d")
	assertBalancedCounts(t, counts, "audit", publishes/2, "audit-a", "audit-b")
	assertBalancedCounts(t, counts, "metrics", publishes, "metrics-a")

	for _, group := range stats.Groups {
		if group.Delivered != publishes {
			t.Fatalf("group %q delivered = %d, want %d", group.Group, group.Delivered, publishes)
		}
		if group.Failed != 0 {
			t.Fatalf("group %q failed = %d, want 0", group.Group, group.Failed)
		}
	}
}

func TestMemoryBrokerCloseSubscriptionStopsDelivery(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	var calls int
	sub, err := broker.SubscribeGroup(ctx, "runtime.events", "workers", "worker-a", func(_ context.Context, _ Delivery) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeGroup returned error: %v", err)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := broker.Publish(ctx, "runtime.events", Envelope{Name: "v1.test"}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("handler calls = %d, want 0", calls)
	}

	stats := broker.Stats("runtime.events")
	if stats.GroupCount != 0 {
		t.Fatalf("GroupCount = %d, want 0", stats.GroupCount)
	}
}

func TestEnvelopeFromCoreEventRoundTripPreservesTraceScopeAndTags(t *testing.T) {
	source := coreevents.Event{
		Name:    coreevents.Name("v1.finance.create_transaction"),
		Payload: map[string]any{"amount": 42},
		Context: map[string]any{
			"trace_id": "tr-42",
			"source":   "telegram",
		},
		ChatID: 17,
		Scope:  "business",
		Tags:   []string{"invoice", "vat"},
	}

	envelope := EnvelopeFromCoreEvent(source)
	roundTrip := envelope.CoreEvent()

	if envelope.TraceID != "tr-42" {
		t.Fatalf("Envelope trace id = %q, want tr-42", envelope.TraceID)
	}
	if roundTrip.Name != source.Name {
		t.Fatalf("CoreEvent name = %q, want %q", roundTrip.Name, source.Name)
	}
	if roundTrip.ChatID != source.ChatID || roundTrip.Scope != source.Scope {
		t.Fatalf("CoreEvent scope/chat mismatch: %+v", roundTrip)
	}
	if got, _ := roundTrip.Context["trace_id"].(string); got != "tr-42" {
		t.Fatalf("CoreEvent trace id = %q, want tr-42", got)
	}
	if len(roundTrip.Tags) != 2 || roundTrip.Tags[0] != "invoice" || roundTrip.Tags[1] != "vat" {
		t.Fatalf("CoreEvent tags = %+v, want invoice/vat", roundTrip.Tags)
	}
}

func assertBalancedCounts(t *testing.T, counts map[deliveryCountKey]int, group string, want int, consumers ...string) {
	t.Helper()

	for _, consumer := range consumers {
		got := counts[deliveryCountKey{group: group, consumer: consumer}]
		if got != want {
			t.Fatalf("%s/%s deliveries = %d, want %d", group, consumer, got, want)
		}
	}
}
