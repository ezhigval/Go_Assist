package auth

import "context"

// AuthAPI публичный контракт модуля auth (транспорт и ядро дергают только его).
type AuthAPI interface {
	CreateSession(ctx context.Context, userID string, roles []Role) (token string, err error)
	ValidateToken(ctx context.Context, token string) (*Session, error)
	RevokeSession(ctx context.Context, token string) error
	// CanEmit проверяет, разрешено ли сессии инициировать событие (роли × имя события).
	CanEmit(s *Session, eventName string) bool
	// EnrichContext добавляет user_id и роли в map контекста события.
	EnrichContext(s *Session, ctx map[string]any)
	Start(ctx context.Context) error
	Stop() error
}
