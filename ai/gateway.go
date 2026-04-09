package ai

import "context"

// Gateway шлюз к LLM (заглушка; реализация — HTTP к провайдеру).
type Gateway interface {
	Complete(ctx context.Context, prompt string) (text string, confidence float64, err error)
}

// StubGateway детерминированный stub без сети.
type StubGateway struct{}

// Complete возвращает шаблонное предложение (короткий контекст — низкая уверенность, без лавины событий).
func (StubGateway) Complete(ctx context.Context, prompt string) (string, float64, error) {
	_ = ctx
	if len(prompt) < 48 {
		return "", 0.35, nil
	}
	return "link_todo_to_calendar", 0.78, nil
}
