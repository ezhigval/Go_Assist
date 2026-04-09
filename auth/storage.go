package auth

import "context"

// SessionStore абстракция хранилища сессий (БД/Redis — снаружи модуля).
type SessionStore interface {
	Put(ctx context.Context, token string, s *Session) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
}
