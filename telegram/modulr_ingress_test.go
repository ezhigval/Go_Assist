package telegram

import (
	"context"
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
}

func (f *fakeBotAPI) RegisterCommand(cmd string, h handler.HandlerFunc) error     { return nil }
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
}
