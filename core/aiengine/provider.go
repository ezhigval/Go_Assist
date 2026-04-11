package aiengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

type inferenceProvider interface {
	Infer(ctx context.Context, req Request, models []ModelSpec) ([]Decision, error)
}

type localHTTPProvider struct {
	baseURL string
	client  *http.Client
}

type unsupportedProvider struct {
	name string
}

type inferRequest struct {
	TraceID  string      `json:"trace_id"`
	ChatID   int64       `json:"chat_id"`
	Scope    string      `json:"scope"`
	Text     string      `json:"text"`
	Tags     []string    `json:"tags,omitempty"`
	KindHint string      `json:"kind_hint,omitempty"`
	Models   []ModelSpec `json:"models,omitempty"`
}

type inferResponse struct {
	Decisions []Decision `json:"decisions"`
}

var (
	externalEmailPattern = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	externalPhonePattern = regexp.MustCompile(`\+?\d[\d\-\s()]{7,}\d`)
	externalCardPattern  = regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`)
)

func newInferenceProvider(cfg Config) inferenceProvider {
	switch cfg.Provider {
	case "", "stub":
		return nil
	case "local":
		return &localHTTPProvider{
			baseURL: cfg.ProviderBaseURL,
			client: &http.Client{
				Timeout: cfg.ProviderTimeout,
			},
		}
	default:
		return unsupportedProvider{name: cfg.Provider}
	}
}

func (p *localHTTPProvider) Infer(ctx context.Context, req Request, models []ModelSpec) ([]Decision, error) {
	payload := inferRequest{
		TraceID:  req.TraceID,
		ChatID:   req.ChatID,
		Scope:    req.Scope,
		Text:     redactPIIForExternal(req.Text),
		Tags:     append([]string(nil), req.Tags...),
		KindHint: req.KindHint,
		Models:   append([]ModelSpec(nil), models...),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal infer request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/infer", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build infer request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("perform infer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("infer request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out inferResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode infer response: %w", err)
	}
	return out.Decisions, nil
}

func (p unsupportedProvider) Infer(ctx context.Context, req Request, models []ModelSpec) ([]Decision, error) {
	_ = ctx
	_ = req
	_ = models
	return nil, fmt.Errorf("provider %q is not implemented", p.name)
}

func redactPIIForExternal(text string) string {
	out := externalEmailPattern.ReplaceAllString(text, "<redacted:email>")
	out = externalCardPattern.ReplaceAllString(out, "<redacted:card>")
	out = externalPhonePattern.ReplaceAllString(out, "<redacted:phone>")
	return out
}
