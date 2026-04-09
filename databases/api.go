package databases

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// DatabaseAPI — публичный интерфейс для взаимодействия микросервисов с БД
type DatabaseAPI interface {
	// Пользователи
	GetOrCreateUser(ctx context.Context, tgID int64, username string) (*User, error)
	UpdateUser(ctx context.Context, tgID int64, username string) error

	// Чаты
	GetOrCreateChat(ctx context.Context, tgID int64, title, chatType string) (*Chat, error)

	// Сессии диалогов
	GetSession(ctx context.Context, chatID int64) (*Session, error)
	SetSession(ctx context.Context, chatID int64, state string, payload map[string]interface{}) error
	ClearSession(ctx context.Context, chatID int64) error

	// Статистика
	LogAction(ctx context.Context, userID int64, action string, metadata map[string]interface{}) error
	GetStats(ctx context.Context) (*StatsSummary, error)

	// Универсальные методы (для сложных кастомных запросов других сервисов)
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, query string, args ...interface{}) (pgx.CommandTag, error)

	// Жизненный цикл
	Start(ctx context.Context) error
	Stop() error
}
