package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"modulr/events"
)

// publisher публикует доменные события (интерфейс — без импорта других модулей).
type publisher interface {
	Publish(evt events.Event)
}

// Service реализует AuthAPI.
type Service struct {
	cfg   Config
	store SessionStore
	bus   publisher
}

// NewService собирает модуль auth.
func NewService(cfg Config, store SessionStore, bus publisher) *Service {
	if store == nil {
		store = NewMemorySessionStore()
	}
	return &Service{cfg: cfg, store: store, bus: bus}
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession выдаёт токен и публикует v1.auth.session.created.
func (s *Service) CreateSession(ctx context.Context, userID string, roles []Role) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("auth: empty user id")
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	sess := &Session{
		Token:     token,
		UserID:    userID,
		Roles:     roles,
		CreatedAt: now,
		ExpiresAt: now.Add(s.cfg.SessionTTL),
		Meta:      make(map[string]any),
	}
	if err := s.store.Put(ctx, token, sess); err != nil {
		return "", err
	}
	if s.bus != nil {
		tid := events.TraceIDFromContext(ctx)
		s.bus.Publish(events.Event{
			Name:    events.V1AuthSessionCreated,
			Payload: sess,
			Source:  "auth",
			TraceID: tid,
			Context: map[string]any{"user_id": userID},
		})
	}
	return token, nil
}

// ValidateToken проверяет токен и срок действия.
func (s *Service) ValidateToken(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, fmt.Errorf("auth: empty token")
	}
	sess, err := s.store.Get(ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = s.store.Delete(ctx, token)
		return nil, fmt.Errorf("auth: session expired")
	}
	return sess, nil
}

// RevokeSession инвалидирует сессию.
func (s *Service) RevokeSession(ctx context.Context, token string) error {
	return s.store.Delete(ctx, token)
}

// roleAllows простая матрица: guest — только чтение системных; user — широкий набор; admin — всё.
func (s *Service) roleAllows(r Role, eventName string) bool {
	switch r {
	case RoleAdmin:
		return true
	case RoleGuest:
		return eventName == string(events.V1SystemStartup)
	case RoleUser:
		return true
	default:
		return false
	}
}

// CanEmit true, если хотя бы одна роль разрешает событие.
func (s *Service) CanEmit(sess *Session, eventName string) bool {
	if sess == nil {
		return false
	}
	for _, r := range sess.Roles {
		if s.roleAllows(r, eventName) {
			return true
		}
	}
	return false
}

// EnrichContext прокидывает user_id и роли в контекст шины.
func (s *Service) EnrichContext(sess *Session, ctx map[string]any) {
	if sess == nil || ctx == nil {
		return
	}
	ctx["user_id"] = sess.UserID
	roles := make([]string, 0, len(sess.Roles))
	for _, r := range sess.Roles {
		roles = append(roles, string(r))
	}
	ctx["roles"] = roles
}

// Start логирует готовность; подписки на шину auth не требует — его вызывает транспорт/ядро.
func (s *Service) Start(ctx context.Context) error {
	log.Println("auth: module started")
	return nil
}

// Stop освобождает ресурсы (store внешний — не закрываем).
func (s *Service) Stop() error {
	log.Println("auth: module stopped")
	return nil
}
