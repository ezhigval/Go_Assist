package knowledge

import "modulr/events"

// Article сохранённый материал (заметка, источник, выжимка).
type Article struct {
	events.EntityBase
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Source   string   `json:"source"`
	Topics   []string `json:"topics"`
	Verified bool     `json:"verified"`
}

// QueryResult результат поиска по базе знаний.
type QueryResult struct {
	Query   string    `json:"query"`
	Hits    []Article `json:"hits"`
	Latency int64     `json:"latency_ms"`
}

// FactCheck результат проверки утверждения.
type FactCheck struct {
	events.EntityBase
	Claim    string   `json:"claim"`
	Verdict  string   `json:"verdict"` // confirmed, disputed, unknown
	Score    float64  `json:"score"`
	Evidence []string `json:"evidence_ids"`
}

// TopicGraph узел графа тем и связей.
type TopicGraph struct {
	events.EntityBase
	TopicID string             `json:"topic_id"`
	Label   string             `json:"label"`
	Related map[string]float64 `json:"related"` // id -> weight
}
