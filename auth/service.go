package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
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
	scope, allowedScopes := sessionAccessFromContext(ctx)
	sess := &Session{
		Token:         token,
		UserID:        userID,
		Scope:         scope,
		AllowedScopes: allowedScopes,
		Roles:         roles,
		CreatedAt:     now,
		ExpiresAt:     now.Add(s.cfg.SessionTTL),
		Meta:          make(map[string]any),
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

// ValidateSessionReference проверяет opaque reference без доступа к raw token.
func (s *Service) ValidateSessionReference(ctx context.Context, reference string) (*Session, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, fmt.Errorf("auth: empty session reference")
	}
	refStore, ok := s.store.(SessionReferenceStore)
	if !ok {
		return nil, fmt.Errorf("auth: session reference store is not supported")
	}
	sess, err := refStore.GetByReference(ctx, strings.TrimSpace(strings.ToLower(reference)))
	if err != nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = refStore.DeleteByReference(ctx, reference)
		return nil, fmt.Errorf("auth: session expired")
	}
	return sess, nil
}

// RevokeSession инвалидирует сессию.
func (s *Service) RevokeSession(ctx context.Context, token string) error {
	return s.store.Delete(ctx, token)
}

// RevokeSessionReference инвалидирует сессию по opaque reference.
func (s *Service) RevokeSessionReference(ctx context.Context, reference string) error {
	refStore, ok := s.store.(SessionReferenceStore)
	if !ok {
		return fmt.Errorf("auth: session reference store is not supported")
	}
	if strings.TrimSpace(reference) == "" {
		return nil
	}
	return refStore.DeleteByReference(ctx, strings.TrimSpace(strings.ToLower(reference)))
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

// CanAccessScope true, если auth-сессия разрешает target scope.
func (s *Service) CanAccessScope(sess *Session, targetScope string) bool {
	return ScopeAllowed(sess, targetScope)
}

// AuthorizeEvent объединяет проверку роли и разрешённого scope.
func (s *Service) AuthorizeEvent(sess *Session, eventName, targetScope string) bool {
	return s.CanEmit(sess, eventName) && s.CanAccessScope(sess, targetScope)
}

// EnrichContext прокидывает user_id и роли в контекст шины.
func (s *Service) EnrichContext(sess *Session, ctx map[string]any) {
	if sess == nil || ctx == nil {
		return
	}
	ctx["user_id"] = sess.UserID
	ctx["scope"] = normalizeSessionScope(sess.Scope)
	ctx["allowed_scopes"] = sessionAllowedScopes(sess.Scope, sess.AllowedScopes)
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
