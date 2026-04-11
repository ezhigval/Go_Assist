package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	coreevents "modulr/core/events"
	"modulr/events"
	"modulr/finance"
	"modulr/knowledge"
	"modulr/tracker"
)

const (
	eventTrackerCreateReminder events.Name = "v1.tracker.create_reminder"
	eventTrackerCreateTask     events.Name = "v1.tracker.create_task"
	eventFinanceCreateTxn      events.Name = "v1.finance.create_transaction"
	eventKnowledgeSaveQuery    events.Name = "v1.knowledge.save_query"
	eventKnowledgeSaveNote     events.Name = "v1.knowledge.save_note"
	eventOrchestratorOutcome   events.Name = events.Name(coreevents.V1OrchestratorDecisionOutcome)
	eventOrchestratorFallback  events.Name = events.Name(coreevents.V1OrchestratorFallback)
	outcomeSource              string      = "app/runtime"
	defaultReminderLead                    = 24 * time.Hour
)

type actionHandlerConfig struct {
	bus       *events.Bus
	tracker   tracker.TrackerAPI
	finance   finance.FinanceAPI
	knowledge knowledge.KnowledgeAPI
	now       func() time.Time
}

func registerActionHandlers(cfg actionHandlerConfig) {
	if cfg.bus == nil {
		return
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}

	if cfg.tracker != nil {
		cfg.bus.Subscribe(eventTrackerCreateReminder, cfg.onTrackerReminder)
		cfg.bus.Subscribe(eventTrackerCreateTask, cfg.onTrackerReminder)
	}
	if cfg.finance != nil {
		cfg.bus.Subscribe(eventFinanceCreateTxn, cfg.onFinanceTransaction)
	}
	if cfg.knowledge != nil {
		cfg.bus.Subscribe(eventKnowledgeSaveQuery, cfg.onKnowledgeSave)
		cfg.bus.Subscribe(eventKnowledgeSaveNote, cfg.onKnowledgeSave)
	}
}

func (cfg actionHandlerConfig) onTrackerReminder(evt events.Event) {
	start := cfg.now()
	payload, _ := payloadToMap(evt.Payload)

	title := firstStringFromMap(payload, "title", "note", "text", "query")
	if title == "" {
		title = "Напоминание"
	}
	item := &tracker.CheckListItem{
		Title:    title,
		DueAt:    firstTime(payload["due_at"], payload["due"], start.Add(defaultReminderLead)),
		Done:     false,
		Source:   "orchestrator",
		LinkedID: stringFromAny(evt.Context["decision_id"]),
	}
	item.Context = segmentFromEvent(evt, payload)
	item.Tags = append([]string{"orchestrator"}, stringSlice(payload["tags"])...)

	err := cfg.tracker.AddChecklistItem(context.Background(), item)
	cfg.publishOutcome(evt, start, err)
}

func (cfg actionHandlerConfig) onFinanceTransaction(evt events.Event) {
	start := cfg.now()
	payload, _ := payloadToMap(evt.Payload)

	tx := &finance.Transaction{
		Type:            finance.TransactionType(firstNonEmptyString(stringFromAny(payload["type"]), string(finance.TransactionExpense))),
		AmountMinor:     int64FromAny(payload["amount_minor"]),
		Currency:        firstNonEmptyString(stringFromAny(payload["currency"]), "RUB"),
		Category:        stringFromAny(payload["category"]),
		Counterparty:    stringFromAny(payload["counterparty"]),
		Memo:            firstStringFromMap(payload, "memo", "title", "text", "note"),
		LinkedEntityIDs: stringSlice(payload["linked_entity_ids"]),
	}
	tx.Context = segmentFromEvent(evt, payload)
	tx.Tags = append([]string{"orchestrator"}, stringSlice(payload["tags"])...)

	err := cfg.finance.CreateTransaction(context.Background(), tx)
	cfg.publishOutcome(evt, start, err)
}

func (cfg actionHandlerConfig) onKnowledgeSave(evt events.Event) {
	start := cfg.now()
	payload, _ := payloadToMap(evt.Payload)

	body := firstStringFromMap(payload, "text", "note", "query", "body")
	if body == "" {
		body = fmt.Sprint(evt.Payload)
	}
	article := &knowledge.Article{
		Title:    titleFromText(firstStringFromMap(payload, "title"), body),
		Body:     body,
		Source:   "orchestrator",
		Topics:   stringSlice(payload["topics"]),
		Verified: false,
	}
	article.Context = segmentFromEvent(evt, payload)
	article.Tags = append([]string{"orchestrator"}, stringSlice(payload["tags"])...)

	err := cfg.knowledge.SaveArticle(context.Background(), article)
	cfg.publishOutcome(evt, start, err)
}

func (cfg actionHandlerConfig) publishOutcome(evt events.Event, startedAt time.Time, err error) {
	if cfg.bus == nil {
		return
	}

	scope := string(segmentFromEvent(evt, nil))
	if scope == "" {
		scope = string(events.DefaultSegment())
	}
	cfg.bus.Publish(events.Event{
		Name: eventOrchestratorOutcome,
		Payload: map[string]any{
			"model_id":     stringFromAny(evt.Context["model_id"]),
			"decision_id":  stringFromAny(evt.Context["decision_id"]),
			"action_event": string(evt.Name),
			"ok":           err == nil,
			"error":        errorString(err),
			"latency_ms":   time.Since(startedAt).Milliseconds(),
			"scope":        scope,
		},
		Source:  outcomeSource,
		TraceID: evt.TraceID,
		Context: map[string]any{
			"trace_id": evt.TraceID,
			"scope":    scope,
			"chat_id":  evt.Context["chat_id"],
		},
	})
}

func segmentFromEvent(evt events.Event, payload map[string]any) events.Segment {
	if seg := events.ParseSegmentFromAny(evt.Context["scope"]); seg != "" {
		return seg
	}
	if seg := events.ParseSegmentFromAny(evt.Context["context"]); seg != "" {
		return seg
	}
	if seg := events.ParseSegmentFromAny(evt.Context["segment"]); seg != "" {
		return seg
	}
	if seg := events.ParseSegmentFromAny(payload["context"]); seg != "" {
		return seg
	}
	if seg := events.ParseSegmentFromAny(payload["scope"]); seg != "" {
		return seg
	}
	return events.DefaultSegment()
}

func payloadToMap(payload any) (map[string]any, bool) {
	if payload == nil {
		return map[string]any{}, true
	}
	if m, ok := payload.(map[string]any); ok {
		cp := make(map[string]any, len(m))
		for k, v := range m {
			cp[k] = v
		}
		return cp, true
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false
	}
	return out, true
}

func firstStringFromMap(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := strings.TrimSpace(stringFromAny(m[key])); s != "" {
			return s
		}
	}
	return ""
}

func firstTime(values ...any) time.Time {
	for _, value := range values {
		switch v := value.(type) {
		case time.Time:
			if !v.IsZero() {
				return v
			}
		case string:
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				return ts
			}
		}
	}
	return time.Time{}
}

func titleFromText(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		runes := []rune(candidate)
		if len(runes) > 64 {
			return string(runes[:64])
		}
		return candidate
	}
	return "Запрос пользователя"
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
	default:
		return 0
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
