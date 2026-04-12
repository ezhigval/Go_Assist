package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"databases"
	"modulr/app"
	modulrauth "modulr/auth"
	telegrampkg "telegram"
	"telegram/state"
)

type persistenceBundle struct {
	store       state.Store
	auth        modulrauth.AuthAPI
	runtimeOpts []app.RuntimeOption
	close       func() error
}

func initPersistence(parent context.Context, cfg telegrampkg.Config) (persistenceBundle, error) {
	switch strings.ToLower(cfg.StateStore) {
	case "", "memory":
		authSvc := modulrauth.NewService(modulrauth.LoadConfig(), modulrauth.NewMemorySessionStore(), nil)
		return persistenceBundle{
			store: state.NewMemoryStore(),
			auth:  authSvc,
			close: authSvc.Stop,
		}, nil
	case "postgres", "database", "db":
		ctx, cancel := context.WithTimeout(parent, 10*time.Second)
		defer cancel()

		db, err := databases.InitDB(ctx, databases.LoadConfig())
		if err != nil {
			return persistenceBundle{}, fmt.Errorf("init postgres persistence: %w", err)
		}
		if err := db.Start(ctx); err != nil {
			_ = db.Stop()
			return persistenceBundle{}, fmt.Errorf("start postgres persistence: %w", err)
		}

		sessionRepo := databaseSessionRepository{db: db}
		authSvc := modulrauth.NewService(modulrauth.LoadConfig(), databases.NewAuthSessionStore(db), nil)
		return persistenceBundle{
			store: state.NewDatabaseStore(sessionRepo),
			auth:  authSvc,
			runtimeOpts: []app.RuntimeOption{
				app.WithEventJournal(databaseRuntimeJournal{db: db}),
			},
			close: func() error {
				if err := authSvc.Stop(); err != nil {
					_ = db.Stop()
					return err
				}
				return db.Stop()
			},
		}, nil
	default:
		return persistenceBundle{}, fmt.Errorf("unsupported TELEGRAM_STATE_STORE %q", cfg.StateStore)
	}
}

type sessionDatabase interface {
	GetOrCreateChat(ctx context.Context, tgID int64, title, chatType string) (*databases.Chat, error)
	GetSession(ctx context.Context, chatID int64) (*databases.Session, error)
	SetSession(ctx context.Context, chatID int64, state string, payload map[string]interface{}) error
	ClearSession(ctx context.Context, chatID int64) error
}

type journalDatabase interface {
	AppendJournalEvent(ctx context.Context, entry databases.EventJournalEntry) (*databases.EventJournalEntry, error)
}

type databaseSessionRepository struct {
	db sessionDatabase
}

func (r databaseSessionRepository) EnsureChat(ctx context.Context, chatID int64) error {
	// Минимальный bootstrap записи чата под FK sessions.chat_id -> chats.tg_id.
	_, err := r.db.GetOrCreateChat(ctx, chatID, "", "private")
	return err
}

func (r databaseSessionRepository) GetSession(ctx context.Context, chatID int64) (state.StoredSession, error) {
	sess, err := r.db.GetSession(ctx, chatID)
	if err != nil {
		return state.StoredSession{}, err
	}
	return state.StoredSession{
		Key:     sess.State,
		Payload: copyPayload(sess.Payload),
	}, nil
}

func (r databaseSessionRepository) SetSession(ctx context.Context, chatID int64, session state.StoredSession) error {
	return r.db.SetSession(ctx, chatID, session.Key, copyPayload(session.Payload))
}

func (r databaseSessionRepository) ClearSession(ctx context.Context, chatID int64) error {
	return r.db.ClearSession(ctx, chatID)
}

type databaseRuntimeJournal struct {
	db journalDatabase
}

func (j databaseRuntimeJournal) WriteEvent(ctx context.Context, record app.JournalRecord) error {
	_, err := j.db.AppendJournalEvent(ctx, databases.EventJournalEntry{
		TraceID:   record.TraceID,
		ChatID:    record.ChatID,
		Scope:     record.Scope,
		EventName: record.EventName,
		Status:    record.Status,
		Source:    record.Source,
		Payload:   copyAnyMap(record.Payload),
		Metadata:  copyAnyMap(record.Metadata),
	})
	return err
}

func copyPayload(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyAnyMap(src map[string]any) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
