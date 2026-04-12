package telegram

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"modulr/app"
	modulrauth "modulr/auth"
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

type fakeAuthAPI struct {
	createToken string
	createErr   error
	validateErr error
	revokeErr   error
	session     *modulrauth.Session

	lastCreateUserID string
	lastCreateRoles  []modulrauth.Role
	lastReference    string
	lastContext      map[string]any
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

func (f *fakeAuthAPI) CreateSession(ctx context.Context, userID string, roles []modulrauth.Role) (string, error) {
	f.lastCreateUserID = userID
	f.lastCreateRoles = append([]modulrauth.Role(nil), roles...)
	if f.createErr != nil {
		return "", f.createErr
	}
	if f.createToken != "" {
		return f.createToken, nil
	}
	return "transport-token", nil
}

func (f *fakeAuthAPI) ValidateToken(ctx context.Context, token string) (*modulrauth.Session, error) {
	return nil, fmt.Errorf("unexpected ValidateToken(%q)", token)
}

func (f *fakeAuthAPI) ValidateSessionReference(ctx context.Context, reference string) (*modulrauth.Session, error) {
	f.lastReference = reference
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	if f.session != nil {
		clone := *f.session
		clone.AllowedScopes = append([]string(nil), f.session.AllowedScopes...)
		clone.Roles = append([]modulrauth.Role(nil), f.session.Roles...)
		return &clone, nil
	}
	return &modulrauth.Session{
		UserID:        "42",
		Scope:         "personal",
		AllowedScopes: []string{"personal"},
		Roles:         []modulrauth.Role{modulrauth.RoleUser},
		ExpiresAt:     time.Now().Add(time.Hour),
	}, nil
}

func (f *fakeAuthAPI) RevokeSession(ctx context.Context, token string) error {
	return fmt.Errorf("unexpected RevokeSession(%q)", token)
}

func (f *fakeAuthAPI) RevokeSessionReference(ctx context.Context, reference string) error {
	f.lastReference = reference
	return f.revokeErr
}

func (f *fakeAuthAPI) CanEmit(s *modulrauth.Session, eventName string) bool { return true }

func (f *fakeAuthAPI) EnrichContext(s *modulrauth.Session, ctx map[string]any) {
	f.lastContext = ctx
	ctx["user_id"] = s.UserID
	ctx["roles"] = []string{"user"}
	ctx["allowed_scopes"] = append([]string(nil), s.AllowedScopes...)
}

func (f *fakeAuthAPI) Start(ctx context.Context) error { return nil }
func (f *fakeAuthAPI) Stop() error                     { return nil }

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

func TestRegisterModulrIngressLoginCommandStoresAuthReference(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{}
	authAPI := &fakeAuthAPI{
		createToken: "issued-token",
		session: &modulrauth.Session{
			UserID:        "6",
			Scope:         "business",
			AllowedScopes: []string{"business"},
			Roles:         []modulrauth.Role{modulrauth.RoleUser},
			ExpiresAt:     time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC),
		},
	}

	if err := RegisterModulrIngress(api, rt, "business", time.Second, WithAuth(authAPI, AuthConfig{})); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	loginHandler := api.commands["login"]
	if loginHandler == nil {
		t.Fatalf("expected /login handler to be registered")
	}

	resp, err := loginHandler(context.Background(), &handler.Request{
		ChatID:   5,
		UserID:   6,
		Username: "alice",
		Text:     "/login",
		State:    state.SetActiveScope(state.Session{}, "business"),
	})
	if err != nil {
		t.Fatalf("login handler returned error: %v", err)
	}
	if resp == nil || !strings.Contains(resp.Text, "Auth session активна") {
		t.Fatalf("unexpected login response: %+v", resp)
	}
	if got := state.AuthSessionReference(resp.NextState); got != modulrauth.SessionReference("issued-token") {
		t.Fatalf("expected auth reference to be stored, got %q", got)
	}
	if authAPI.lastCreateUserID != "6" {
		t.Fatalf("CreateSession userID = %q, want 6", authAPI.lastCreateUserID)
	}
}

func TestRegisterModulrIngressInjectsAuthContextIntoRuntime(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{
		result: app.HandleResult{
			TraceID: "tr-auth",
			Status:  "completed",
		},
	}
	authAPI := &fakeAuthAPI{
		session: &modulrauth.Session{
			UserID:        "auth-user-1",
			Scope:         "travel",
			AllowedScopes: []string{"travel"},
			Roles:         []modulrauth.Role{modulrauth.RoleUser},
			ExpiresAt:     time.Now().Add(time.Hour),
		},
	}

	if err := RegisterModulrIngress(api, rt, "travel", time.Second, WithAuth(authAPI, AuthConfig{})); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	resp, err := api.textHandler(context.Background(), &handler.Request{
		ChatID:   7,
		UserID:   8,
		Username: "bob",
		Text:     "покажи поездки",
		State:    state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, "travel"), "ref-123"),
	})
	if err != nil {
		t.Fatalf("text handler returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if got := rt.msg.Context["user_id"]; got != "auth-user-1" {
		t.Fatalf("runtime context user_id = %v, want auth-user-1", got)
	}
	scopes, ok := rt.msg.Context["allowed_scopes"].([]string)
	if !ok || len(scopes) != 1 || scopes[0] != "travel" {
		t.Fatalf("runtime context allowed_scopes = %v", rt.msg.Context["allowed_scopes"])
	}
}

func TestRegisterModulrIngressRequiresAuthWhenConfigured(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{
		result: app.HandleResult{
			TraceID: "tr-noauth",
			Status:  "completed",
		},
	}
	authAPI := &fakeAuthAPI{}

	if err := RegisterModulrIngress(api, rt, "personal", time.Second, WithAuth(authAPI, AuthConfig{Required: true})); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	resp, err := api.textHandler(context.Background(), &handler.Request{
		ChatID:   9,
		UserID:   10,
		Username: "carol",
		Text:     "создай задачу",
		State:    state.SetActiveScope(state.Session{}, "personal"),
	})
	if err != nil {
		t.Fatalf("text handler returned error: %v", err)
	}
	if resp == nil || !strings.Contains(resp.Text, "/login") {
		t.Fatalf("expected auth prompt, got %+v", resp)
	}
	if rt.msg.Text != "" {
		t.Fatalf("runtime should not be called when auth is required, got %+v", rt.msg)
	}
}

func TestRegisterModulrIngressMarksRuntimeContextAsAuthRequired(t *testing.T) {
	api := &fakeBotAPI{}
	rt := &fakeRuntime{
		result: app.HandleResult{
			TraceID: "tr-auth-required",
			Status:  "completed",
		},
	}
	authAPI := &fakeAuthAPI{
		session: &modulrauth.Session{
			UserID:        "auth-user-2",
			Scope:         "business",
			AllowedScopes: []string{"business"},
			Roles:         []modulrauth.Role{modulrauth.RoleUser},
		},
	}

	if err := RegisterModulrIngress(api, rt, "personal", time.Second, WithAuth(authAPI, AuthConfig{Required: true})); err != nil {
		t.Fatalf("RegisterModulrIngress returned error: %v", err)
	}

	resp, err := api.textHandler(context.Background(), &handler.Request{
		ChatID:   11,
		UserID:   12,
		Username: "dora",
		Text:     "создай задачу",
		State:    state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, "business"), "ref-777"),
	})
	if err != nil {
		t.Fatalf("text handler returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if got := rt.msg.Context["auth_required"]; got != true {
		t.Fatalf("runtime context auth_required = %v, want true", got)
	}
}
