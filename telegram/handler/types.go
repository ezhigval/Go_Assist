package handler

import (
	"context"
	"telegram/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Request — стандартизированный запрос от пользователя
type Request struct {
	ChatID   int64
	UserID   int64
	Username string
	Text     string
	Command  string
	Callback string
	State    state.Session
	Files    []tgbotapi.FileConfig
}

// Response — стандартизированный ответ для отправки в Telegram
type Response struct {
	Text      string
	ParseMode string
	Keyboard  tgbotapi.ReplyMarkup // может быть Reply или Inline
	Edit      bool                 // редактировать последнее сообщение
	Delete    bool                 // удалить сообщение
	NextState state.Session        // обновить состояние
}

// HandlerFunc — сигнатура бизнес-обработчика
type HandlerFunc func(ctx context.Context, req *Request) (*Response, error)
