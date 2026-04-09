package aiengine

import (
	"strings"
	"sync"
)

// ModelRouter выбирает подходящие модели по типу запроса и области (Scope).
type ModelRouter struct {
	mu     sync.RWMutex
	models map[string]ModelSpec
}

// NewModelRouter создаёт роутер с дефолтным набором заглушек.
func NewModelRouter() *ModelRouter {
	r := &ModelRouter{models: make(map[string]ModelSpec)}
	r.seedDefaults()
	return r
}

func (r *ModelRouter) seedDefaults() {
	r.mu.Lock()
	defer r.mu.Unlock()
	defaults := []ModelSpec{
		{ID: "llm_primary", Kind: "llm", Priority: 10, Weight: 1.0, Capabilities: []string{"nlp", "intent", "general"}, Enabled: true},
		{ID: "route_planner", Kind: "route_planner", Priority: 20, Weight: 1.0, Capabilities: []string{"maps", "geo", "route"}, Enabled: true},
		{ID: "finance_analyzer", Kind: "finance_analyzer", Priority: 15, Weight: 1.0, Capabilities: []string{"finance", "invoice", "budget"}, Enabled: true},
		{ID: "schedule_optimizer", Kind: "schedule_optimizer", Priority: 15, Weight: 1.0, Capabilities: []string{"calendar", "time", "reminder"}, Enabled: true},
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range defaults {
		r.models[m.ID] = m
	}
}

// Upsert обновляет или добавляет модель.
func (r *ModelRouter) Upsert(spec ModelSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[spec.ID] = spec
}

// Select возвращает отсортированный список моделей, релевантных запросу.
func (r *ModelRouter) Select(req Request) []ModelSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	text := strings.ToLower(req.Text + " " + req.KindHint + " " + strings.Join(req.Tags, " "))
	var out []ModelSpec
	for _, m := range r.models {
		if !m.Enabled {
			continue
		}
		if matchesCapabilities(text, m.Capabilities) || m.Kind == "llm" {
			out = append(out, m)
		}
	}
	// простая сортировка: приоритет выше — раньше
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Priority > out[i].Priority {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func matchesCapabilities(text string, caps []string) bool {
	for _, c := range caps {
		if c != "" && strings.Contains(text, c) {
			return true
		}
	}
	return false
}
