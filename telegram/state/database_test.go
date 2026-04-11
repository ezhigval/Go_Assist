package state

import (
	"context"
	"errors"
	"testing"
)

func TestDatabaseStoreGetNormalizesIdleState(t *testing.T) {
	store := NewDatabaseStore(&fakeSessionRepo{
		session: StoredSession{
			Key:     "idle",
			Payload: map[string]interface{}{"draft": "note", "_active_scope": "travel"},
		},
	})

	got := store.Get(context.Background(), 42)
	if got.Key != "" {
		t.Fatalf("expected idle session to normalize to empty key, got %q", got.Key)
	}
	if got.Payload["draft"] != "note" {
		t.Fatalf("expected idle payload to be preserved, got %+v", got.Payload)
	}
	if ActiveScope(got) != "travel" {
		t.Fatalf("expected active scope to survive normalization, got %+v", got.Payload)
	}
}

func TestDatabaseStoreSetEnsuresChatAndPersistsSession(t *testing.T) {
	repo := &fakeSessionRepo{}
	store := NewDatabaseStore(repo)

	err := store.Set(context.Background(), 99, Session{
		Key: "awaiting_input",
		Payload: map[string]interface{}{
			"step": float64(2),
		},
	})
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if repo.ensureChatCalls != 1 {
		t.Fatalf("EnsureChat calls = %d, want 1", repo.ensureChatCalls)
	}
	if repo.saved.Key != "awaiting_input" {
		t.Fatalf("saved key = %q, want awaiting_input", repo.saved.Key)
	}
	if repo.saved.Payload["step"] != float64(2) {
		t.Fatalf("saved payload = %+v", repo.saved.Payload)
	}
}

func TestDatabaseStoreClearReturnsRepositoryError(t *testing.T) {
	store := NewDatabaseStore(&fakeSessionRepo{clearErr: errors.New("clear failed")})

	err := store.Clear(context.Background(), 12)
	if err == nil || err.Error() != "clear failed" {
		t.Fatalf("expected clear error, got %v", err)
	}
}

type fakeSessionRepo struct {
	session         StoredSession
	saved           StoredSession
	getErr          error
	setErr          error
	clearErr        error
	ensureChatErr   error
	ensureChatCalls int
}

func (f *fakeSessionRepo) EnsureChat(ctx context.Context, chatID int64) error {
	f.ensureChatCalls++
	return f.ensureChatErr
}

func (f *fakeSessionRepo) GetSession(ctx context.Context, chatID int64) (StoredSession, error) {
	return f.session, f.getErr
}

func (f *fakeSessionRepo) SetSession(ctx context.Context, chatID int64, session StoredSession) error {
	f.saved = session
	return f.setErr
}

func (f *fakeSessionRepo) ClearSession(ctx context.Context, chatID int64) error {
	return f.clearErr
}
