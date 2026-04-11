package app

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	coreevents "modulr/core/events"
	"modulr/tracker"
)

func TestRuntimeHandleMessageCreatesTrackerReminder(t *testing.T) {
	rt := NewRuntime()
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() {
		if err := rt.Stop(ctx); err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	}()

	traceID, err := rt.HandleMessage(ctx, InboundMessage{
		ChatID:   42,
		UserID:   1001,
		Username: "demo",
		Text:     "напоминание купить молоко после работы",
		Scope:    "personal",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	if traceID == "" {
		t.Fatalf("HandleMessage returned empty trace id")
	}

	item := waitForChecklistItem(t, rt, "tracker:check:")
	if item.Context != "personal" {
		t.Fatalf("checklist context = %q, want personal", item.Context)
	}
	if !strings.Contains(strings.ToLower(item.Title), "напомин") && !strings.Contains(strings.ToLower(item.Title), "купить") {
		t.Fatalf("unexpected checklist title %q", item.Title)
	}
	if item.Source != "orchestrator" {
		t.Fatalf("checklist source = %q, want orchestrator", item.Source)
	}

	stats, err := rt.Orchestrator().GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if stats.ErrorCount != 0 {
		t.Fatalf("unexpected orchestrator errors: %d", stats.ErrorCount)
	}
	if len(stats.ChatHistories[42]) == 0 {
		t.Fatalf("expected chat history to be recorded")
	}
}

func TestRuntimeHandleMessageSyncReturnsCompletedOutcome(t *testing.T) {
	rt := NewRuntime()
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() {
		if err := rt.Stop(ctx); err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	}()

	result, err := rt.HandleMessageSync(ctx, InboundMessage{
		ChatID:   73,
		UserID:   2002,
		Username: "sync",
		Text:     "напоминание позвонить клиенту",
		Scope:    "business",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("unexpected result status %q", result.Status)
	}
	if result.ActionEvent != string(EventTrackerCreateReminder) {
		t.Fatalf("unexpected action event %q", result.ActionEvent)
	}
	if result.ModelID == "" {
		t.Fatalf("expected model_id in result")
	}
	if result.TraceID == "" || result.DecisionID == "" {
		t.Fatalf("expected trace and decision IDs in result: %+v", result)
	}
}

func TestRuntimeHandleMessageSyncWritesJournalEntries(t *testing.T) {
	journal := &fakeJournal{}
	rt := NewRuntime(WithEventJournal(journal))
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() {
		if err := rt.Stop(ctx); err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	}()

	result, err := rt.HandleMessageSync(ctx, InboundMessage{
		ChatID:   88,
		UserID:   5005,
		Username: "journal",
		Text:     "напоминание проверить журнал событий",
		Scope:    "personal",
		Source:   "telegram",
		Tags:     []string{"test"},
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}

	entries := waitForJournalEntries(t, journal, result.TraceID, 2)
	if !containsJournalEvent(entries, string(coreevents.V1MessageReceived), "accepted") {
		t.Fatalf("expected accepted message entry, got %+v", entries)
	}
	if !containsJournalEvent(entries, string(EventOrchestratorOutcome), "completed") {
		t.Fatalf("expected completed outcome entry, got %+v", entries)
	}
}

func waitForChecklistItem(t *testing.T, rt *Runtime, prefix string) tracker.CheckListItem {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		keys, err := rt.Store().ListPrefix(context.Background(), prefix)
		if err != nil {
			t.Fatalf("ListPrefix returned error: %v", err)
		}
		if len(keys) == 0 {
			time.Sleep(25 * time.Millisecond)
			continue
		}

		var item tracker.CheckListItem
		if err := rt.Store().GetJSON(context.Background(), keys[0], &item); err != nil {
			t.Fatalf("GetJSON returned error: %v", err)
		}
		return item
	}

	t.Fatalf("timed out waiting for checklist item")
	return tracker.CheckListItem{}
}

func waitForJournalEntries(t *testing.T, journal *fakeJournal, traceID string, want int) []JournalRecord {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entries := journal.EntriesByTrace(traceID)
		if len(entries) >= want {
			return entries
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for journal entries trace=%s", traceID)
	return nil
}

func containsJournalEvent(entries []JournalRecord, eventName, status string) bool {
	for _, entry := range entries {
		if entry.EventName == eventName && entry.Status == status {
			return true
		}
	}
	return false
}

type fakeJournal struct {
	mu      sync.Mutex
	entries []JournalRecord
}

func (f *fakeJournal) WriteEvent(ctx context.Context, record JournalRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, JournalRecord{
		TraceID:   record.TraceID,
		ChatID:    record.ChatID,
		Scope:     record.Scope,
		EventName: record.EventName,
		Status:    record.Status,
		Source:    record.Source,
		Payload:   cloneContext(record.Payload),
		Metadata:  cloneContext(record.Metadata),
		CreatedAt: record.CreatedAt,
	})
	return nil
}

func (f *fakeJournal) EntriesByTrace(traceID string) []JournalRecord {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []JournalRecord
	for _, entry := range f.entries {
		if entry.TraceID == traceID {
			out = append(out, entry)
		}
	}
	return out
}
