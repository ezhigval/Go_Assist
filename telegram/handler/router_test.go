package handler

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram/state"
)

func TestRouterHandleReturnsResponseAndStoresState(t *testing.T) {
	store := state.NewMemoryStore()
	router := NewRouter(store)
	router.RegisterText("", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			Text:      "ok",
			NextState: state.NewSession("awaiting_input"),
		}, nil
	})

	resp, err := router.Handle(context.Background(), tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      "hello",
			Chat:      &tgbotapi.Chat{ID: 11},
			From:      &tgbotapi.User{ID: 12, UserName: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if resp == nil || resp.Text != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if session := store.Get(context.Background(), 11); session.Key != "awaiting_input" {
		t.Fatalf("state was not persisted: %+v", session)
	}
}

func TestRouterHandleReturnsStoreError(t *testing.T) {
	router := NewRouter(failingStore{err: errors.New("persist failed")})
	router.RegisterText("", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			Text:      "ok",
			NextState: state.NewSession("awaiting_input"),
		}, nil
	})

	_, err := router.Handle(context.Background(), tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      "hello",
			Chat:      &tgbotapi.Chat{ID: 11},
			From:      &tgbotapi.User{ID: 12, UserName: "demo"},
		},
	})
	if err == nil || err.Error() != "persist failed" {
		t.Fatalf("expected persist error, got %v", err)
	}
}

func TestRouterHandlePreservesPayloadOnlyState(t *testing.T) {
	store := state.NewMemoryStore()
	router := NewRouter(store)
	router.RegisterText("", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Text: "ok"}, nil
	})

	if err := store.Set(context.Background(), 11, state.SetActiveScope(state.Session{}, "business")); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	resp, err := router.Handle(context.Background(), tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      "hello",
			Chat:      &tgbotapi.Chat{ID: 11},
			From:      &tgbotapi.User{ID: 12, UserName: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if got := state.ActiveScope(store.Get(context.Background(), 11)); got != "business" {
		t.Fatalf("active scope was not preserved, got %q", got)
	}
}

func TestRouterHandlePreservesAuthSessionReference(t *testing.T) {
	store := state.NewMemoryStore()
	router := NewRouter(store)
	router.RegisterText("", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Text: "ok"}, nil
	})

	seed := state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, "business"), "ref-123")
	if err := store.Set(context.Background(), 11, seed); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	resp, err := router.Handle(context.Background(), tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      "hello",
			Chat:      &tgbotapi.Chat{ID: 11},
			From:      &tgbotapi.User{ID: 12, UserName: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}

	stored := store.Get(context.Background(), 11)
	if got := state.AuthSessionReference(stored); got != "ref-123" {
		t.Fatalf("auth reference was not preserved, got %q", got)
	}
	if got := state.ActiveScope(stored); got != "business" {
		t.Fatalf("active scope was not preserved, got %q", got)
	}
}

type failingStore struct {
	err error
}

func (f failingStore) Get(ctx context.Context, chatID int64) state.Session {
	return state.Session{}
}

func (f failingStore) Set(ctx context.Context, chatID int64, session state.Session) error {
	return f.err
}

func (f failingStore) Clear(ctx context.Context, chatID int64) error {
	return f.err
}
