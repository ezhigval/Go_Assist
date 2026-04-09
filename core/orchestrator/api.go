package orchestrator

import (
	"context"

	coreevents "modulr/core/events"
)

// OrchestratorAPI публичный контракт центрального оркестратора.
type OrchestratorAPI interface {
	// RegisterModule регистрирует доменный модуль и его действия (для валидации AI-решений).
	RegisterModule(ctx context.Context, name string, actions []string) error
	// ProcessEvent обрабатывает входящее событие шины (основной вход).
	ProcessEvent(ctx context.Context, e coreevents.Event) error
	// GetStats возвращает агрегированные метрики и истории (для API мониторинга).
	GetStats(ctx context.Context) (Stats, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
