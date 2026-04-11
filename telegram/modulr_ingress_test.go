package telegram

import (
	"context"
	"strings"
	"testing"
	"time"

	"modulr/app"
	"telegram/handler"
	"telegram/state"
)

type fakeRuntime struct {
	result app.HandleResult
	err    error
	msg    app.InboundMessage
}

func (f *fakeRuntime) HandleMessageSync(ctx context.Context, msg app.InboundMessage) (app.HandleResult, error) {
	f.msg = msg
	return f.result, f.err
}

type fakeBotAPI struct {
	textHandler handler.HandlerFunc
	commands    map[string]handler.HandlerFunc
}

func (f *fakeBotAPI) RegisterCommand(cmd string, h handler.HandlerFunc) error {
	if f.commands == nil {
		f.commands = make(map[string]handler.HandlerFunc)
	}
	f.commands[cmd] = h
	return nil
}
func (f *fakeBotAPI) RegisterCallback(prefix string, h handler.HandlerFunc) error { return nil }
func (f *fakeBotAPI) RegisterState(stateKey string, h handler.HandlerFunc) error  { return nil }
func (f *fakeBotAPI) SendMessage(ctx context.Context, chatID int64, msg *handler.Response) error {
	return nil
}
func (f *fakeBotAPI) EditMessage(ctx context.Context, chatID int64, msgID int, msg *handler.Response) error {
	return nil
}
func (f *fakeBotAPI) GetState(chatID int64) state.Session { return state.Session{} }
func (f *fakeBotAPI) SetState(chatID int64, s state.Session) error {
	return nil
}
func (f *fakeBotAPI) Start(ctx context.Context) error { return nil }
func (f *fakeBotAPI) Stop() error                     { return nil }
func (f *fakeBotAPI) RegisterText(pattern string, h handler.HandlerFunc) error {
	f.textHandler = h
	return nil
}

func TestRegisterModulrIngressFormatsCompletedResult(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{
		result: app.HandleResult{
			TraceID:     "tr-1",
			Status:      "completed",
			ActionEvent: "v1.tracker.create_reminder",
		},
	}

	if err := RegisterModulrIngress(api, rt, "family", time.Second); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}
	if api.textHandler == nil {
		t.Fatalf("expected text handler to be registered")
	}
	if api.commands["scope"] == nil {
		t.Fatalf("expected /scope command to be registered")
	}

	resp, err := api.textHandler(context.Background(), &handler.Request{
		ChatID:   5,
		UserID:   6,
		Username: "alice",
		Text:     "напоминание купить подарок",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp == nil || resp.Text == "" {
		t.Fatalf("expected non-empty response")
	}
	if rt.msg.Scope != "family" || rt.msg.Source != "telegram" {
		t.Fatalf("unexpected ingress message: %+v", rt.msg)
	}
	if got := state.ActiveScope(resp.NextState); got != "family" {
		t.Fatalf("expected response to preserve active scope, got %+v", resp.NextState)
	}
}

func TestRegisterModulrIngressFormatsFallbackTimeoutAndFailedResults(t *testing.T) {
	cases := []struct {
		name   string
		result app.HandleResult
		want   []string
	}{
		{
			name: "fallback",
			result: app.HandleResult{
				TraceID: "tr-fallback",
				Status:  "fallback",
				Reason:  "manual_confirm_or_rules",
			},
			want: []string{"Нужна ручная проверка", "manual_confirm_or_rules", "tr-fallback"},
		},
		{
			name: "timeout",
			result: app.HandleResult{
				TraceID: "tr-timeout",
				Status:  "timeout",
			},
			want: []string{"Запрос принят в обработку", "tr-timeout"},
		},
		{
			name: "failed",
			result: app.HandleResult{
				TraceID: "tr-failed",
				Status:  "failed",
				Error:   "storage error",
			},
			want: []string{"Действие завершилось с ошибкой", "storage error", "tr-failed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeBotAPI{}
			rt := &fakeRuntime{result: tc.result}

			if err := RegisterModulrIngress(api, rt, "personal", time.Second); err != nil {
				t.Fatalf("RegisterModulrIngress returned error: %v", err)
			}

			resp, err := api.textHandler(context.Background(), &handler.Request{
				ChatID:   10,
				UserID:   20,
				Username: "bob",
				Text:     "test",
			})
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if resp == nil {
				t.Fatalf("expected response")
			}
			for _, needle := range tc.want {
				if !strings.Contains(resp.Text, needle) {
					t.Fatalf("response %q does not contain %q", resp.Text, needle)
				}
			}
			if got := state.ActiveScope(resp.NextState); got != "personal" {
				t.Fatalf("expected response to preserve default scope, got %+v", resp.NextState)
			}
		})
	}
}

func TestRegisterModulrIngressUsesPersistedActiveScope(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{
		result: app.HandleResult{
			TraceID: "tr-scope",
			Status:  "completed",
		},
	}

	if err := RegisterModulrIngress(api, rt, "personal", time.Second); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	resp, err := api.textHandler(context.Background(), &handler.Request{
		ChatID:   5,
		UserID:   6,
		Username: "alice",
		Text:     "покажи бюджет",
		State:    state.SetActiveScope(state.Session{}, "business"),
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if rt.msg.Scope != "business" {
		t.Fatalf("expected persisted scope to be used, got %+v", rt.msg)
	}
}

func TestRegisterModulrIngressScopeCommandSwitchesScope(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{}

	if err := RegisterModulrIngress(api, rt, "personal", time.Second); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	scopeHandler := api.commands["scope"]
	if scopeHandler == nil {
		t.Fatalf("expected /scope handler to be registered")
	}

	resp, err := scopeHandler(context.Background(), &handler.Request{
		ChatID: 5,
		Text:   "/scope travel",
		State:  state.SetActiveScope(state.Session{}, "personal"),
	})
	if err != nil {
		t.Fatalf("scope handler returned error: %v", err)
	}
	if resp == nil || !strings.Contains(resp.Text, "travel") {
		t.Fatalf("unexpected scope command response: %+v", resp)
	}
	if got := state.ActiveScope(resp.NextState); got != "travel" {
		t.Fatalf("expected next state to store travel scope, got %+v", resp.NextState)
	}
}
