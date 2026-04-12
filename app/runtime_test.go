package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"modulr/core/aiengine"
	coreevents "modulr/core/events"
	"modulr/finance"
	"modulr/knowledge"
	"modulr/metrics"
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

func TestRuntimeHandleMessageSyncCreatesFinanceTransaction(t *testing.T) {
	rt := NewRuntime(WithAIEngine(&fakeAIEngine{
		decisions: []aiengine.Decision{
			{
				Target:     "finance",
				Action:     "create_transaction",
				ModelID:    "finance-test",
				Confidence: 0.94,
				Parameters: map[string]any{
					"type":         "expense",
					"amount_minor": int64(1599),
					"currency":     "RUB",
					"counterparty": "coffee_shop",
					"memo":         "капучино после встречи",
					"tags":         []string{"coffee", "meeting"},
				},
			},
		},
	}))
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
		ChatID:   91,
		UserID:   3003,
		Username: "finance",
		Text:     "внеси трату на кофе",
		Scope:    "business",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "completed" || result.ActionEvent != string(EventFinanceCreateTxn) {
		t.Fatalf("unexpected result: %+v", result)
	}

	txn := waitForTransaction(t, rt)
	if txn.AmountMinor != 1599 || txn.Currency != "RUB" {
		t.Fatalf("unexpected transaction amounts: %+v", txn)
	}
	if txn.Context != "business" {
		t.Fatalf("transaction context = %q, want business", txn.Context)
	}
	if txn.Counterparty != "coffee_shop" {
		t.Fatalf("transaction counterparty = %q", txn.Counterparty)
	}
}

func TestRuntimeHandleMessageSyncCreatesKnowledgeArticle(t *testing.T) {
	rt := NewRuntime(WithAIEngine(&fakeAIEngine{
		decisions: []aiengine.Decision{
			{
				Target:     "knowledge",
				Action:     "save_note",
				ModelID:    "knowledge-test",
				Confidence: 0.91,
				Parameters: map[string]any{
					"title":  "Итоги встречи",
					"text":   "Клиент подтвердил следующий созвон на вторник.",
					"topics": []string{"sales", "meeting"},
					"tags":   []string{"crm"},
				},
			},
		},
	}))
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
		ChatID:   92,
		UserID:   4004,
		Username: "knowledge",
		Text:     "сохрани заметку о встрече",
		Scope:    "personal",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "completed" || result.ActionEvent != string(EventKnowledgeSaveNote) {
		t.Fatalf("unexpected result: %+v", result)
	}

	article := waitForArticle(t, rt)
	if article.Title != "Итоги встречи" {
		t.Fatalf("article title = %q", article.Title)
	}
	if article.Context != "personal" || article.Source != "orchestrator" {
		t.Fatalf("unexpected article metadata: %+v", article)
	}
	if !strings.Contains(article.Body, "следующий созвон") {
		t.Fatalf("unexpected article body %q", article.Body)
	}
}

func TestRuntimeHandleMessageSyncRejectsCrossScopeDecisionWithoutExplicitPolicy(t *testing.T) {
	journal := &fakeJournal{}
	rt := NewRuntime(
		WithEventJournal(journal),
		WithAIEngine(&fakeAIEngine{
			decisions: []aiengine.Decision{
				{
					Target:     "finance",
					Action:     "create_transaction",
					ModelID:    "finance-cross-scope",
					Confidence: 0.96,
					Scope:      "business",
					Parameters: map[string]any{"memo": "не должно пройти"},
				},
			},
		}),
	)
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
		ChatID:   95,
		UserID:   7007,
		Username: "scope-guard",
		Text:     "запиши расход в бизнес",
		Scope:    "personal",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "fallback" {
		t.Fatalf("unexpected result status %q", result.Status)
	}
	if !strings.Contains(result.Reason, "no decisions passed filters") {
		t.Fatalf("unexpected fallback reason %q", result.Reason)
	}

	entries := waitForJournalEntries(t, journal, result.TraceID, 2)
	if !containsJournalEvent(entries, string(EventOrchestratorFallback), "fallback") {
		t.Fatalf("expected fallback journal entry, got %+v", entries)
	}
}

func TestRuntimeHandleMessageSyncAllowsCrossScopeDecisionWithExplicitPolicy(t *testing.T) {
	rt := NewRuntime(WithAIEngine(&fakeAIEngine{
		decisions: []aiengine.Decision{
			{
				Target:     "finance",
				Action:     "create_transaction",
				ModelID:    "finance-cross-scope",
				Confidence: 0.96,
				Scope:      "business",
				Parameters: map[string]any{
					"type":         "expense",
					"amount_minor": int64(2500),
					"currency":     "RUB",
					"memo":         "разрешённый cross-scope расход",
				},
			},
		},
	}))
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
		ChatID:   96,
		UserID:   8008,
		Username: "scope-override",
		Text:     "запиши расход для бизнеса",
		Scope:    "personal",
		Source:   "telegram",
		Context: map[string]any{
			"allowed_scopes": []string{"business"},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "completed" || result.ActionEvent != string(EventFinanceCreateTxn) {
		t.Fatalf("unexpected result: %+v", result)
	}

	txn := waitForTransaction(t, rt)
	if txn.Context != "business" {
		t.Fatalf("transaction context = %q, want business", txn.Context)
	}
}

func TestRuntimeHandleMessageSyncRejectsDecisionByRoleAuthorization(t *testing.T) {
	rt := NewRuntime(WithAIEngine(&fakeAIEngine{
		decisions: []aiengine.Decision{
			{
				Target:     "tracker",
				Action:     "create_reminder",
				ModelID:    "tracker-guest-denied",
				Confidence: 0.96,
				Scope:      "personal",
				Parameters: map[string]any{"title": "не должно пройти"},
			},
		},
	}))
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
		ChatID:   101,
		UserID:   9001,
		Username: "guest-user",
		Text:     "создай напоминание",
		Scope:    "personal",
		Source:   "telegram",
		Context: map[string]any{
			"roles":         []string{"guest"},
			"auth_required": true,
		},
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "fallback" {
		t.Fatalf("unexpected result status %q", result.Status)
	}
	if !strings.Contains(result.Reason, "role_denied") {
		t.Fatalf("unexpected fallback reason %q", result.Reason)
	}
}

func TestRuntimeHandleMessageSyncAllowsDecisionForUserRole(t *testing.T) {
	rt := NewRuntime(WithAIEngine(&fakeAIEngine{
		decisions: []aiengine.Decision{
			{
				Target:     "tracker",
				Action:     "create_reminder",
				ModelID:    "tracker-user-allowed",
				Confidence: 0.96,
				Scope:      "personal",
				Parameters: map[string]any{"title": "должно пройти"},
			},
		},
	}))
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
		ChatID:   102,
		UserID:   9002,
		Username: "auth-user",
		Text:     "создай напоминание",
		Scope:    "personal",
		Source:   "telegram",
		Context: map[string]any{
			"roles":         []string{"user"},
			"auth_required": true,
			"user_id":       "auth-user-102",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "completed" || result.ActionEvent != string(EventTrackerCreateReminder) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRuntimeMetricsCaptureScopeAndTrace(t *testing.T) {
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
		ChatID:   97,
		UserID:   9009,
		Username: "metrics",
		Text:     "напоминание записать trace",
		Scope:    "travel",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}

	snapshot := waitForRuntimeMetrics(t, rt, result.TraceID)
	if snapshot.ScopeCounts["travel"] == 0 {
		t.Fatalf("expected travel scope in metrics snapshot, got %+v", snapshot.ScopeCounts)
	}
	trace := findMetricsTrace(snapshot.Traces, result.TraceID)
	if trace.TraceID == "" {
		t.Fatalf("expected trace summary for %s, got %+v", result.TraceID, snapshot.Traces)
	}
	if trace.Scope != "travel" {
		t.Fatalf("trace scope = %q, want travel", trace.Scope)
	}
	if trace.EventCount < 2 {
		t.Fatalf("expected multiple events in trace summary, got %+v", trace)
	}
}

func TestRuntimePreservesAuthContextAlongsideTransportIdentity(t *testing.T) {
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

	traceID, err := rt.HandleMessage(ctx, InboundMessage{
		ChatID:   98,
		UserID:   9010,
		Username: "transport-user",
		Text:     "проверь auth metadata",
		Scope:    "business",
		Source:   "telegram",
		Context: map[string]any{
			"user_id":        "auth-user-42",
			"roles":          []string{"admin"},
			"allowed_scopes": []string{"business", "travel"},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	entries := waitForJournalEntries(t, journal, traceID, 1)
	entry := entries[0]
	if got := entry.Metadata["user_id"]; got != "auth-user-42" {
		t.Fatalf("journal metadata user_id = %v, want auth-user-42", got)
	}
	if got := entry.Metadata["transport_user_id"]; got != int64(9010) {
		t.Fatalf("journal metadata transport_user_id = %v, want 9010", got)
	}
	if got := entry.Metadata["transport_username"]; got != "transport-user" {
		t.Fatalf("journal metadata transport_username = %v, want transport-user", got)
	}
}

func TestRuntimeHandleMessageSyncReturnsFallbackWhenAIAnalyzeFails(t *testing.T) {
	journal := &fakeJournal{}
	rt := NewRuntime(
		WithEventJournal(journal),
		WithAIEngine(&fakeAIEngine{err: errors.New("provider unavailable")}),
	)
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
		ChatID:   93,
		UserID:   5005,
		Username: "fallback",
		Text:     "напоминание проверить провайдера",
		Scope:    "personal",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "fallback" {
		t.Fatalf("unexpected result status %q", result.Status)
	}
	if !strings.Contains(result.Reason, "provider unavailable") {
		t.Fatalf("unexpected fallback reason %q", result.Reason)
	}

	entries := waitForJournalEntries(t, journal, result.TraceID, 2)
	if !containsJournalEvent(entries, string(EventOrchestratorFallback), "fallback") {
		t.Fatalf("expected fallback journal entry, got %+v", entries)
	}
}

func TestRuntimeHandleMessageSyncReturnsTimeoutForUnhandledCalendarAction(t *testing.T) {
	journal := &fakeJournal{}
	rt := NewRuntime(
		WithEventJournal(journal),
		WithAIEngine(&fakeAIEngine{
			decisions: []aiengine.Decision{
				{
					Target:     "calendar",
					Action:     "create_event",
					ModelID:    "calendar-test",
					Confidence: 0.97,
					Parameters: map[string]any{"title": "Созвон с командой"},
				},
			},
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() {
		if err := rt.Stop(context.Background()); err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	}()

	result, err := rt.HandleMessageSync(ctx, InboundMessage{
		ChatID:   94,
		UserID:   6006,
		Username: "timeout",
		Text:     "создай встречу на завтра",
		Scope:    "business",
		Source:   "telegram",
	})
	if err != nil {
		t.Fatalf("HandleMessageSync returned error: %v", err)
	}
	if result.Status != "timeout" {
		t.Fatalf("unexpected result status %q", result.Status)
	}

	entries := waitForJournalEntries(t, journal, result.TraceID, 2)
	if !containsJournalEvent(entries, string(EventTransportResponseTimeout), "timeout") {
		t.Fatalf("expected timeout journal entry, got %+v", entries)
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

func waitForTransaction(t *testing.T, rt *Runtime) finance.Transaction {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		keys, err := rt.Store().ListPrefix(context.Background(), "finance:transaction:")
		if err != nil {
			t.Fatalf("ListPrefix returned error: %v", err)
		}
		if len(keys) == 0 {
			time.Sleep(25 * time.Millisecond)
			continue
		}

		var txn finance.Transaction
		if err := rt.Store().GetJSON(context.Background(), keys[0], &txn); err != nil {
			t.Fatalf("GetJSON returned error: %v", err)
		}
		return txn
	}

	t.Fatalf("timed out waiting for transaction")
	return finance.Transaction{}
}

func waitForArticle(t *testing.T, rt *Runtime) knowledge.Article {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		keys, err := rt.Store().ListPrefix(context.Background(), "knowledge:article:")
		if err != nil {
			t.Fatalf("ListPrefix returned error: %v", err)
		}
		if len(keys) == 0 {
			time.Sleep(25 * time.Millisecond)
			continue
		}

		var article knowledge.Article
		if err := rt.Store().GetJSON(context.Background(), keys[0], &article); err != nil {
			t.Fatalf("GetJSON returned error: %v", err)
		}
		return article
	}

	t.Fatalf("timed out waiting for article")
	return knowledge.Article{}
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

func waitForRuntimeMetrics(t *testing.T, rt *Runtime, traceID string) metrics.Snapshot {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := rt.Metrics().Snapshot(10)
		if trace := findMetricsTrace(snapshot.Traces, traceID); trace.TraceID != "" {
			return snapshot
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for runtime metrics trace=%s", traceID)
	return metrics.Snapshot{}
}

func findMetricsTrace(traces []metrics.TraceSummary, traceID string) metrics.TraceSummary {
	for _, trace := range traces {
		if trace.TraceID == traceID {
			return trace
		}
	}
	return metrics.TraceSummary{}
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

type fakeAIEngine struct {
	mu        sync.Mutex
	decisions []aiengine.Decision
	err       error
	feedbacks []aiengine.Feedback
}

func (f *fakeAIEngine) Analyze(ctx context.Context, req aiengine.Request) ([]aiengine.Decision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	out := make([]aiengine.Decision, len(f.decisions))
	for i, decision := range f.decisions {
		if decision.ID == "" {
			decision.ID = fmt.Sprintf("fake_decision_%d", i+1)
		}
		if decision.Scope == "" {
			decision.Scope = req.Scope
		}
		if decision.CreatedAt.IsZero() {
			decision.CreatedAt = now
		}
		if decision.Parameters == nil {
			decision.Parameters = map[string]any{}
		}
		out[i] = decision
	}
	return out, nil
}

func (f *fakeAIEngine) RegisterModel(ctx context.Context, spec aiengine.ModelSpec) error {
	_ = ctx
	_ = spec
	return nil
}

func (f *fakeAIEngine) Feedback(ctx context.Context, fb aiengine.Feedback) error {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	f.feedbacks = append(f.feedbacks, fb)
	return nil
}

func (f *fakeAIEngine) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

func (f *fakeAIEngine) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}
