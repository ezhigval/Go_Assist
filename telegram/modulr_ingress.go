package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"modulr/app"
	"telegram/handler"
)

// RuntimeIngress контракт root runtime для handoff из transport-слоя.
type RuntimeIngress interface {
	HandleMessageSync(ctx context.Context, msg app.InboundMessage) (app.HandleResult, error)
}

// RegisterModulrIngress подключает Telegram text flow к runtime корневого модуля.
func RegisterModulrIngress(api BotAPI, ingress RuntimeIngress, defaultScope string, timeout time.Duration) error {
	if ingress == nil {
		return fmt.Errorf("telegram: nil runtime ingress")
	}
	if defaultScope == "" {
		defaultScope = "personal"
	}
	if timeout <= 0 {
		timeout = 3500 * time.Millisecond
	}

	return api.RegisterText("", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		text := strings.TrimSpace(req.Text)
		if text == "" {
			return &handler.Response{Text: "Пока работаю только с текстовыми сообщениями."}, nil
		}

		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		result, err := ingress.HandleMessageSync(runCtx, app.InboundMessage{
			ChatID:   req.ChatID,
			UserID:   req.UserID,
			Username: req.Username,
			Text:     text,
			Scope:    defaultScope,
			Source:   "telegram",
		})
		if err != nil {
			return nil, err
		}

		return &handler.Response{
			Text:      formatModulrResult(result),
			ParseMode: "Markdown",
		}, nil
	})
}

func formatModulrResult(result app.HandleResult) string {
	switch result.Status {
	case "completed":
		return fmt.Sprintf("Принял запрос и выполнил действие: *%s*\nTrace: `%s`", humanAction(result.ActionEvent), result.TraceID)
	case "fallback":
		return fmt.Sprintf("Нужна ручная проверка.\nПричина: `%s`\nTrace: `%s`", fallbackReason(result.Reason), result.TraceID)
	case "timeout":
		return fmt.Sprintf("Запрос принят в обработку.\nTrace: `%s`", result.TraceID)
	case "failed":
		return fmt.Sprintf("Действие завершилось с ошибкой: `%s`\nTrace: `%s`", fallbackReason(result.Error), result.TraceID)
	default:
		return fmt.Sprintf("Запрос принят.\nTrace: `%s`", result.TraceID)
	}
}

func humanAction(name string) string {
	return app.HumanAction(name)
}

func fallbackReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "неизвестно"
	}
	return reason
}
