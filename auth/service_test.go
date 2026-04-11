package auth

import (
	"context"
	"testing"
	"time"
)

func TestCreateSessionUsesScopeContext(t *testing.T) {
	t.Parallel()

	store := &captureSessionStore{}
	svc := NewService(Config{SessionTTL: time.Hour}, store, nil)

	ctx := WithAllowedScopes(WithSessionScope(context.Background(), "business"), []string{"travel", "business", "unknown"})
	token, err := svc.CreateSession(ctx, "user-1", []Role{RoleUser})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected token to be returned")
	}
	if store.session == nil {
		t.Fatal("expected store to capture session")
	}
	if store.session.Scope != "business" {
		t.Fatalf("scope = %q, want business", store.session.Scope)
	}
	wantScopes := []string{"business", "travel"}
	if len(store.session.AllowedScopes) != len(wantScopes) {
		t.Fatalf("allowed scopes = %v, want %v", store.session.AllowedScopes, wantScopes)
	}
	for i, scope := range wantScopes {
		if store.session.AllowedScopes[i] != scope {
			t.Fatalf("allowed scopes = %v, want %v", store.session.AllowedScopes, wantScopes)
		}
	}
}

func TestAuthorizeEventChecksScope(t *testing.T) {
	t.Parallel()

	svc := NewService(Config{SessionTTL: time.Hour}, NewMemorySessionStore(), nil)
	sess := &Session{
		UserID:        "user-1",
		Scope:         "personal",
		AllowedScopes: []string{"personal", "business"},
		Roles:         []Role{RoleUser},
	}

	if !svc.AuthorizeEvent(sess, "v1.finance.transaction.created", "business") {
		t.Fatal("expected business scope to be allowed")
	}
	if svc.AuthorizeEvent(sess, "v1.finance.transaction.created", "pets") {
		t.Fatal("expected pets scope to be denied")
	}
}

func TestEnrichContextAddsScopeMetadata(t *testing.T) {
	t.Parallel()

	svc := NewService(Config{SessionTTL: time.Hour}, NewMemorySessionStore(), nil)
	ctxMap := map[string]any{}

	svc.EnrichContext(&Session{
		UserID:        "user-1",
		Scope:         "family",
		AllowedScopes: []string{"travel", "family"},
		Roles:         []Role{RoleAdmin},
	}, ctxMap)

	if got := ctxMap["scope"]; got != "family" {
		t.Fatalf("scope = %v, want family", got)
	}
	scopes, ok := ctxMap["allowed_scopes"].([]string)
	if !ok {
		t.Fatalf("allowed_scopes has unexpected type %T", ctxMap["allowed_scopes"])
	}
	wantScopes := []string{"family", "travel"}
	for i, scope := range wantScopes {
		if scopes[i] != scope {
			t.Fatalf("allowed_scopes = %v, want %v", scopes, wantScopes)
		}
	}
}

type captureSessionStore struct {
	session *Session
}

func (c *captureSessionStore) Put(_ context.Context, _ string, s *Session) error {
	c.session = cloneSession(s)
	return nil
}

func (c *captureSessionStore) Get(_ context.Context, _ string) (*Session, error) {
	return cloneSession(c.session), nil
}

func (c *captureSessionStore) Delete(_ context.Context, _ string) error {
	c.session = nil
	return nil
}
