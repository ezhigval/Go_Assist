package aiengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineAnalyzeUsesLocalProviderWithPIIRedaction(t *testing.T) {
	t.Setenv("AI_PROVIDER", "local")
	t.Setenv("AI_PROVIDER_BASE_URL", "http://placeholder.invalid")
	t.Setenv("AI_ALLOW_STUB_FALLBACK", "false")

	var captured inferRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infer" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(inferResponse{
			Decisions: []Decision{
				{
					Target:     "finance",
					Action:     "create_transaction",
					Confidence: 0.91,
					Parameters: map[string]any{"amount_minor": 1200},
				},
			},
		})
	}))
	defer server.Close()
	t.Setenv("AI_PROVIDER_BASE_URL", server.URL)

	engine := NewEngine()
	decs, err := engine.Analyze(context.Background(), Request{
		TraceID: "tr-1",
		ChatID:  10,
		Scope:   "personal",
		Text:    "Напомни про карту 4111 1111 1111 1111 и почту alice@example.com",
		Tags:    []string{"finance"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Target != "finance" || decs[0].Action != "create_transaction" {
		t.Fatalf("unexpected decision %+v", decs[0])
	}
	if captured.Text == "" {
		t.Fatalf("expected provider request text to be captured")
	}
	if captured.Text == "Напомни про карту 4111 1111 1111 1111 и почту alice@example.com" {
		t.Fatalf("expected PII redaction before provider call")
	}
	if !strings.Contains(captured.Text, "<redacted:card>") || !strings.Contains(captured.Text, "<redacted:email>") {
		t.Fatalf("expected redacted tokens in %q", captured.Text)
	}
}

func TestEngineAnalyzeFallsBackToStubWhenProviderFails(t *testing.T) {
	t.Setenv("AI_PROVIDER", "local")
	t.Setenv("AI_PROVIDER_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("AI_ALLOW_STUB_FALLBACK", "true")

	engine := NewEngine()
	decs, err := engine.Analyze(context.Background(), Request{
		TraceID: "tr-2",
		ChatID:  11,
		Scope:   "personal",
		Text:    "напоминание купить молоко",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(decs) == 0 {
		t.Fatalf("expected stub fallback decisions")
	}
	if decs[0].Target != "tracker" || decs[0].Action != "create_reminder" {
		t.Fatalf("unexpected fallback decision %+v", decs[0])
	}
}

func TestEngineAnalyzeReturnsProviderErrorWhenFallbackDisabled(t *testing.T) {
	t.Setenv("AI_PROVIDER", "openai")
	t.Setenv("AI_ALLOW_STUB_FALLBACK", "false")

	engine := NewEngine()
	if _, err := engine.Analyze(context.Background(), Request{
		TraceID: "tr-3",
		ChatID:  12,
		Scope:   "personal",
		Text:    "напоминание оплатить подписку",
	}); err == nil {
		t.Fatalf("expected provider error when fallback is disabled")
	}
}
