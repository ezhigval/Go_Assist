package examples

import (
	"context"
	"telegram/handler"
	"telegram/keyboard"
)

// RegisterStartHandler регистрирует обработчик /start
// Этот файл демонстрирует, как подключать бизнес-логику к боту
func RegisterStartHandler(api BotAPI) {
	api.RegisterCommand("start", handleStart)
}

func handleStart(ctx context.Context, req *handler.Request) (*handler.Response, error) {
	text := "👋 *Добро пожаловать!*\n\n" +
		"Я универсальный Telegram-бот. Используйте меню ниже для навигации.\n" +
		"🔹 _Модули подключаются как плагины_\n" +
		"🔹 _Состояния диалогов управляются автоматически_"

	kb := keyboard.MainReplyKeyboard()

	// STUB: User registration requires publishing v1.metrics.log_metric or v1.user.registered to main bus with scope/tags, non-blocking bot response.
	// STUB: Menu loading requires config/navigation service by user scope through MenuProvider interface.

	return &handler.Response{
		Text:      text,
		ParseMode: "Markdown",
		Keyboard:  kb,
		NextState: handler.StateSession{}, // сброс состояния
	}, nil
}
