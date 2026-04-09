package events

import "context"

// Storage универсальное JSON-хранилище модулей (реализация — снаружи: БД, файлы).
type Storage interface {
	GetJSON(ctx context.Context, key string, dest any) error
	PutJSON(ctx context.Context, key string, v any) error
	Delete(ctx context.Context, key string) error
	ListPrefix(ctx context.Context, prefix string) ([]string, error)
}
