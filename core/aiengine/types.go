package aiengine

import "time"

// Request входной запрос на анализ (без сырого хранения персональных данных в долгоживущем виде).
type Request struct {
	TraceID  string         `json:"trace_id"`
	ChatID   int64          `json:"chat_id"`
	Scope    string         `json:"scope"` // см. modulr/events: personal, family, work, business, health, travel, pets, assets
	Text     string         `json:"text"`
	Tags     []string       `json:"tags,omitempty"`
	KindHint string         `json:"kind_hint,omitempty"` // calendar, finance, route, ...
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Decision атомарное решение для исполнения доменным модулем.
type Decision struct {
	ID         string         `json:"id"`
	Target     string         `json:"target"` // модуль: calendar, tracker, maps, finance, ...
	Action     string         `json:"action"` // глагол: create_event, create_reminder, ...
	Parameters map[string]any `json:"parameters,omitempty"`
	Confidence float64        `json:"confidence"`
	Scope      string         `json:"scope"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Feedback результат исполнения решения (для подстройки весов моделей).
type Feedback struct {
	ModelID    string `json:"model_id"`
	DecisionID string `json:"decision_id"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	Scope      string `json:"scope"`
}

// ModelSpec описание зарегистрированной модели/роутера.
type ModelSpec struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"` // llm, route_planner, finance_analyzer, schedule_optimizer
	Priority     int      `json:"priority"`
	Weight       float64  `json:"weight"`
	Capabilities []string `json:"capabilities"`
	Enabled      bool     `json:"enabled"`
}
