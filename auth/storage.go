package auth

import "context"

// SessionStore абстракция хранилища сессий (БД/Redis — снаружи модуля).
type SessionStore interface {
	Put(ctx context.Context, token string, s *Session) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
}

// SessionReferenceStore хранит доступ к сессиям по безопасному opaque reference.
type SessionReferenceStore interface {
	GetByReference(ctx context.Context, reference string) (*Session, error)
	DeleteByReference(ctx context.Context, reference string) error
}
