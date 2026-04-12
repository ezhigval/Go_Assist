package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"modulr/app"
	"modulr/events"
	"telegram/handler"
	"telegram/state"
)

// RuntimeIngress контракт root runtime для handoff из transport-слоя.
type RuntimeIngress interface {
	HandleMessageSync(ctx context.Context, msg app.InboundMessage) (app.HandleResult, error)
}

// RegisterModulrIngress подключает Telegram text flow к runtime корневого модуля.
func RegisterModulrIngress(api BotAPI, ingress RuntimeIngress, defaultScope string, timeout time.Duration, opts ...IngressOption) error {
	if ingress == nil {
		return fmt.Errorf("telegram: nil runtime ingress")
	}
	defaultScope = normalizeTelegramScope(defaultScope)
	if timeout <= 0 {
		timeout = 3500 * time.Millisecond
	}
	cfg := buildIngressConfig(opts...)

	if err := registerTelegramAuthCommands(api, cfg, defaultScope); err != nil {
		return err
	}

	if err := api.RegisterCommand("scope", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		currentScope := activeTelegramScope(req.State, defaultScope)
		nextScope := parseScopeCommand(req.Text)
		if nextScope == "" {
			return &handler.Response{
				Text:      fmt.Sprintf("Активный scope: `%s`\nДоступные scope: `%s`\nИспользование: `/scope <segment>`", currentScope, strings.Join(scopeNames(), "`, `")),
				ParseMode: "Markdown",
				NextState: state.SetActiveScope(state.Session{}, currentScope),
			}, nil
		}

		if events.ParseSegmentFromAny(nextScope) == "" {
			return &handler.Response{
				Text:      fmt.Sprintf("Неизвестный scope: `%s`\nДоступные scope: `%s`", nextScope, strings.Join(scopeNames(), "`, `")),
				ParseMode: "Markdown",
				NextState: state.SetActiveScope(state.Session{}, currentScope),
			}, nil
		}

		nextScope = normalizeTelegramScope(nextScope)
		return &handler.Response{
			Text:      fmt.Sprintf("Активный scope переключён на `%s`", nextScope),
			ParseMode: "Markdown",
			NextState: state.SetActiveScope(state.Session{}, nextScope),
		}, nil
	}); err != nil {
		return err
	}

	return api.RegisterText("", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		text := strings.TrimSpace(req.Text)
		if text == "" {
			return &handler.Response{Text: "Пока работаю только с текстовыми сообщениями."}, nil
		}
		activeScope := activeTelegramScope(req.State, defaultScope)
		authResult, err := authorizeTelegramIngress(ctx, req, activeScope, cfg)
		if err != nil {
			return nil, err
		}
		if authResult.Response != nil {
			authResult.Response.NextState = mergeTelegramState(authResult.NextState, authResult.Response.NextState)
			authResult.Response.NextState = state.SetActiveScope(authResult.Response.NextState, activeScope)
			return authResult.Response, nil
		}

		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		msgContext := make(map[string]any)
		if authResult.Session != nil && cfg.auth != nil {
			cfg.auth.EnrichContext(authResult.Session, msgContext)
		}
		if cfg.authCfg.Required {
			msgContext["auth_required"] = true
		}
		result, err := ingress.HandleMessageSync(runCtx, app.InboundMessage{
			ChatID:   req.ChatID,
			UserID:   req.UserID,
			Username: req.Username,
			Text:     text,
			Scope:    activeScope,
			Source:   "telegram",
			Context:  msgContext,
		})
		if err != nil {
			return nil, err
		}

		nextState := state.SetActiveScope(authResult.NextState, activeScope)
		return &handler.Response{
			Text:      formatModulrResult(result),
			ParseMode: "Markdown",
			NextState: nextState,
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

func activeTelegramScope(session state.Session, defaultScope string) string {
	if scope := state.ActiveScope(session); scope != "" {
		return scope
	}
	return normalizeTelegramScope(defaultScope)
}

func normalizeTelegramScope(scope string) string {
	if seg := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(scope))); seg != "" {
		return string(seg)
	}
	return string(events.DefaultSegment())
}

func parseScopeCommand(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(fields[1]))
}

func scopeNames() []string {
	items := events.AllSegments()
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, string(item))
	}
	return out
}
