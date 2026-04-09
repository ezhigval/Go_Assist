package telegram

import (
	"context"
	"telegram/handler"
	"telegram/state"
)

// BotAPI — публичный интерфейс для взаимодействия с ботом из других микросервисов
type BotAPI interface {
	// Регистрация обработчиков
	RegisterCommand(cmd string, h handler.HandlerFunc) error
	RegisterText(pattern string, h handler.HandlerFunc) error
	RegisterCallback(prefix string, h handler.HandlerFunc) error
	RegisterState(stateKey string, h handler.HandlerFunc) error

	// Прямая отправка сообщений (для рассылок, уведомлений, webhook-триггеров)
	SendMessage(ctx context.Context, chatID int64, msg *handler.Response) error
	EditMessage(ctx context.Context, chatID int64, msgID int, msg *handler.Response) error

	// Управление состоянием диалога
	GetState(chatID int64) state.Session
	SetState(chatID int64, s state.Session) error

	// Жизненный цикл
	Start(ctx context.Context) error
	Stop() error
}
