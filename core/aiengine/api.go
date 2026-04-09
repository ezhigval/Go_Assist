package aiengine

import "context"

// AIEngine фасад ИИ-хаба: анализ, регистрация моделей, обратная связь, жизненный цикл.
type AIEngine interface {
	// Analyze формирует набор решений по запросу (таймауты через ctx).
	Analyze(ctx context.Context, req Request) ([]Decision, error)
	// RegisterModel регистрирует или обновляет спецификацию модели.
	RegisterModel(ctx context.Context, spec ModelSpec) error
	// Feedback принимает исход исполнения решения для обновления весов.
	Feedback(ctx context.Context, fb Feedback) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
