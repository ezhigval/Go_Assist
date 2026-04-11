package metrics

import (
	"context"
	"testing"
	"time"

	"modulr/events"
)

func TestServiceSnapshotAggregatesScopeAndTrace(t *testing.T) {
	t.Parallel()

	bus := events.NewBus(0)
	svc := NewService(Config{}, bus, nil)
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	bus.Publish(events.Event{
		Name:    events.Name("v1.message.received"),
		TraceID: "tr-1",
		Source:  "telegram",
		Context: map[string]any{"scope": "personal"},
	})
	bus.Publish(events.Event{
		Name:    events.Name("v1.finance.create_transaction"),
		TraceID: "tr-1",
		Source:  "app/runtime",
		Context: map[string]any{"scope": "personal"},
	})
	bus.Publish(events.Event{
		Name:    events.Name("v1.knowledge.save_query"),
		TraceID: "tr-2",
		Source:  "app/runtime",
		Context: map[string]any{"scope": "business"},
	})

	snapshot := waitForMetricsSnapshot(t, svc, 2)
	if snapshot.Counts["__total__"] != 3 {
		t.Fatalf("unexpected total counts: %+v", snapshot.Counts)
	}
	if snapshot.ScopeCounts["personal"] != 2 || snapshot.ScopeCounts["business"] != 1 {
		t.Fatalf("unexpected scope counts: %+v", snapshot.ScopeCounts)
	}
	if len(snapshot.Traces) < 2 {
		t.Fatalf("expected at least 2 trace summaries, got %+v", snapshot.Traces)
	}

	trace1 := findTrace(snapshot.Traces, "tr-1")
	if trace1.EventCount != 2 || trace1.Scope != "personal" {
		t.Fatalf("unexpected trace tr-1 summary: %+v", trace1)
	}
	if trace1.LastEvent != "v1.finance.create_transaction" && trace1.LastEvent != "v1.message.received" {
		t.Fatalf("unexpected last event for tr-1: %+v", trace1)
	}

	trace2 := findTrace(snapshot.Traces, "tr-2")
	if trace2.EventCount != 1 || trace2.Scope != "business" {
		t.Fatalf("unexpected trace tr-2 summary: %+v", trace2)
	}
}

func waitForMetricsSnapshot(t *testing.T, svc *Service, wantTraces int) Snapshot {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := svc.Snapshot(10)
		if snapshot.Counts["__total__"] >= 3 && len(snapshot.Traces) >= wantTraces {
			return snapshot
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for metrics snapshot")
	return Snapshot{}
}

func findTrace(traces []TraceSummary, traceID string) TraceSummary {
	for _, trace := range traces {
		if trace.TraceID == traceID {
			return trace
		}
	}
	return TraceSummary{}
}
