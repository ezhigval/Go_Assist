package orchestrator

import (
	"fmt"
	"sync"
)

// ModuleRegistry реестр доменных модулей и их эндпоинтов (действий).
type ModuleRegistry struct {
	mu      sync.RWMutex
	modules map[string]map[string]struct{}
}

// NewModuleRegistry создаёт пустой реестр.
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{modules: make(map[string]map[string]struct{})}
}

// RegisterModule регистрирует модуль и список поддерживаемых действий.
func (r *ModuleRegistry) RegisterModule(name string, actions []string) error {
	if name == "" {
		return fmt.Errorf("orchestrator/registry: empty module name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.modules[name]; !ok {
		r.modules[name] = make(map[string]struct{})
	}
	for _, a := range actions {
		if a == "" {
			continue
		}
		r.modules[name][a] = struct{}{}
	}
	return nil
}

// HasEndpoint проверяет, объявлен ли эндпоинт у модуля.
func (r *ModuleRegistry) HasEndpoint(module, action string) bool {
	if module == "" || action == "" {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	acts, ok := r.modules[module]
	if !ok {
		return false
	}
	_, ok = acts[action]
	return ok
}

// ListModules возвращает копию имён модулей (для мониторинга).
func (r *ModuleRegistry) ListModules() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.modules))
	for k := range r.modules {
		out = append(out, k)
	}
	return out
}
