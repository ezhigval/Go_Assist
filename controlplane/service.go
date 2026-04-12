package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"modulr/events"
)

var (
	// ErrScopeNotFound означает неизвестный scope preset.
	ErrScopeNotFound = errors.New("controlplane: scope not found")
	// ErrScopeConflict означает конфликт scope id после patch/create.
	ErrScopeConflict = errors.New("controlplane: scope conflict")
	// ErrLastScope означает попытку удалить последний доступный scope.
	ErrLastScope = errors.New("controlplane: cannot delete last scope")
	// ErrInvalidScope означает неизвестный segment.
	ErrInvalidScope = errors.New("controlplane: invalid scope")
	// ErrModuleNotFound означает неизвестный module id.
	ErrModuleNotFound = errors.New("controlplane: module not found")
	// ErrPluginNotFound означает неизвестный plugin id.
	ErrPluginNotFound = errors.New("controlplane: plugin not found")
	// ErrBrokerNotFound означает неизвестный broker id.
	ErrBrokerNotFound = errors.New("controlplane: broker not found")
)

var _ API = (*Service)(nil)

// Service хранит in-memory projection operator-конфигурации.
type Service struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

// NewService создаёт control-plane projection со snapshot по умолчанию.
func NewService() *Service {
	return &Service{
		snapshot: defaultSnapshot(),
	}
}

// Health подтверждает готовность in-memory projection.
func (s *Service) Health(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// Snapshot возвращает полный control-plane snapshot.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	if ctx.Err() != nil {
		return Snapshot{}, ctx.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.snapshot), nil
}

// ListScopes возвращает scope presets в текущем порядке.
func (s *Service) ListScopes(ctx context.Context) ([]ScopePreset, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneScopes(s.snapshot.Scopes), nil
}

// CreateScope добавляет новый scope preset или возвращает существующий при совпадении ключа.
func (s *Service) CreateScope(ctx context.Context, scope ScopePreset) (ScopePreset, error) {
	if ctx.Err() != nil {
		return ScopePreset{}, ctx.Err()
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return ScopePreset{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := ScopeKey(scope)
	if existing, index := findScopeLocked(s.snapshot.Scopes, id); index >= 0 {
		return cloneScope(existing), nil
	}

	s.snapshot.Scopes = append(s.snapshot.Scopes, scope)
	s.touchLocked()
	return cloneScope(scope), nil
}

// UpdateScopeTags обновляет tags у существующего preset.
func (s *Service) UpdateScopeTags(ctx context.Context, id string, tags []string) (ScopePreset, error) {
	if ctx.Err() != nil {
		return ScopePreset{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope, index := findScopeLocked(s.snapshot.Scopes, id)
	if index < 0 {
		return ScopePreset{}, ErrScopeNotFound
	}

	updated := scope
	updated.Tags = normalizeTags(tags)
	updatedID := ScopeKey(updated)
	if updatedID != id {
		if _, conflictIndex := findScopeLocked(s.snapshot.Scopes, updatedID); conflictIndex >= 0 {
			return ScopePreset{}, ErrScopeConflict
		}
	}

	s.snapshot.Scopes[index] = updated
	s.touchLocked()
	return cloneScope(updated), nil
}

// DeleteScope удаляет preset, но не позволяет оставить систему без scope.
func (s *Service) DeleteScope(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, index := findScopeLocked(s.snapshot.Scopes, id)
	if index < 0 {
		return ErrScopeNotFound
	}
	if len(s.snapshot.Scopes) <= 1 {
		return ErrLastScope
	}

	s.snapshot.Scopes = append(s.snapshot.Scopes[:index], s.snapshot.Scopes[index+1:]...)
	s.touchLocked()
	return nil
}

// UpdateModule обновляет operator-facing module-control.
func (s *Service) UpdateModule(ctx context.Context, id string, patch ModulePatch) (ModuleControl, error) {
	if ctx.Err() != nil {
		return ModuleControl{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, module := range s.snapshot.Modules {
		if module.ID != id {
			continue
		}
		if patch.Enabled != nil {
			module.Enabled = *patch.Enabled
		}
		if patch.DispatchMode != nil {
			if !patch.DispatchMode.valid() {
				return ModuleControl{}, fmt.Errorf("controlplane: invalid module dispatch mode %q", *patch.DispatchMode)
			}
			module.DispatchMode = *patch.DispatchMode
		}
		if patch.ConsumerGroup != nil {
			module.ConsumerGroup = *patch.ConsumerGroup
		}
		if patch.AllowedScopes != nil {
			if !validScopeList(*patch.AllowedScopes) {
				return ModuleControl{}, ErrInvalidScope
			}
			module.AllowedScopes = cloneStrings(*patch.AllowedScopes)
		}
		if patch.Tags != nil {
			module.Tags = normalizeTags(*patch.Tags)
		}
		if patch.LatencyBudgetMS != nil && *patch.LatencyBudgetMS > 0 {
			module.LatencyBudgetMS = *patch.LatencyBudgetMS
		}
		s.snapshot.Modules[i] = module
		s.touchLocked()
		return cloneModule(module), nil
	}

	return ModuleControl{}, ErrModuleNotFound
}

// UpdatePlugin обновляет operator-facing plugin rollout state.
func (s *Service) UpdatePlugin(ctx context.Context, id string, patch PluginPatch) (PluginControl, error) {
	if ctx.Err() != nil {
		return PluginControl{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, plugin := range s.snapshot.Plugins {
		if plugin.ID != id {
			continue
		}
		if patch.Status != nil {
			if !patch.Status.valid() {
				return PluginControl{}, fmt.Errorf("controlplane: invalid plugin status %q", *patch.Status)
			}
			plugin.Status = *patch.Status
		}
		if patch.Description != nil {
			plugin.Description = *patch.Description
		}
		if patch.Capabilities != nil {
			capabilities := cloneCapabilities(*patch.Capabilities)
			if !validCapabilities(capabilities) {
				return PluginControl{}, ErrInvalidScope
			}
			plugin.Capabilities = capabilities
		}
		s.snapshot.Plugins[i] = plugin
		s.touchLocked()
		return clonePlugin(plugin), nil
	}

	return PluginControl{}, ErrPluginNotFound
}

// CycleBrokerMode переводит lane по memory -> nats -> kafka -> memory.
func (s *Service) CycleBrokerMode(ctx context.Context, id string) (BrokerLane, error) {
	if ctx.Err() != nil {
		return BrokerLane{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, broker := range s.snapshot.Brokers {
		if broker.ID != id {
			continue
		}
		broker.Mode, broker.Status = rotateBrokerMode(broker.Mode)
		s.snapshot.Brokers[i] = broker
		s.touchLocked()
		return cloneBroker(broker), nil
	}

	return BrokerLane{}, ErrBrokerNotFound
}

func (s *Service) touchLocked() {
	s.snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

func normalizeScope(scope ScopePreset) (ScopePreset, error) {
	scope.Segment = string(events.ParseSegmentFromAny(scope.Segment))
	if !validScope(scope.Segment) {
		return ScopePreset{}, ErrInvalidScope
	}
	scope.Tags = normalizeTags(scope.Tags)
	scope.Metadata = cloneAnyMap(scope.Metadata)
	if scope.Metadata == nil {
		scope.Metadata = map[string]any{"source": "v2-control-plane"}
	}
	return scope, nil
}

func validScopeList(scopes []string) bool {
	for _, scope := range scopes {
		if !validScope(scope) {
			return false
		}
	}
	return true
}

func validCapabilities(capabilities []PluginCapability) bool {
	for _, capability := range capabilities {
		if capability.Module == "" || len(capability.Actions) == 0 {
			return false
		}
		if !validScopeList(capability.Scopes) {
			return false
		}
	}
	return true
}

func findScopeLocked(scopes []ScopePreset, id string) (ScopePreset, int) {
	for i, scope := range scopes {
		if ScopeKey(scope) == id {
			return scope, i
		}
	}
	return ScopePreset{}, -1
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	return Snapshot{
		UpdatedAt:  snapshot.UpdatedAt,
		Scopes:     cloneScopes(snapshot.Scopes),
		TagPresets: cloneStrings(snapshot.TagPresets),
		Brokers:    cloneBrokers(snapshot.Brokers),
		Modules:    cloneModules(snapshot.Modules),
		Plugins:    clonePlugins(snapshot.Plugins),
	}
}

func cloneScopes(scopes []ScopePreset) []ScopePreset {
	if len(scopes) == 0 {
		return nil
	}
	out := make([]ScopePreset, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, cloneScope(scope))
	}
	return out
}

func cloneScope(scope ScopePreset) ScopePreset {
	return ScopePreset{
		Segment:  scope.Segment,
		Tags:     cloneStrings(scope.Tags),
		Metadata: cloneAnyMap(scope.Metadata),
	}
}

func cloneBrokers(brokers []BrokerLane) []BrokerLane {
	if len(brokers) == 0 {
		return nil
	}
	out := make([]BrokerLane, 0, len(brokers))
	for _, broker := range brokers {
		out = append(out, cloneBroker(broker))
	}
	return out
}

func cloneBroker(broker BrokerLane) BrokerLane {
	return BrokerLane{
		ID:             broker.ID,
		Title:          broker.Title,
		Topic:          broker.Topic,
		Mode:           broker.Mode,
		Status:         broker.Status,
		Notes:          broker.Notes,
		ConsumerGroups: cloneConsumerGroups(broker.ConsumerGroups),
	}
}

func cloneConsumerGroups(groups []BrokerConsumerGroup) []BrokerConsumerGroup {
	if len(groups) == 0 {
		return nil
	}
	out := make([]BrokerConsumerGroup, len(groups))
	copy(out, groups)
	return out
}

func cloneModules(modules []ModuleControl) []ModuleControl {
	if len(modules) == 0 {
		return nil
	}
	out := make([]ModuleControl, 0, len(modules))
	for _, module := range modules {
		out = append(out, cloneModule(module))
	}
	return out
}

func cloneModule(module ModuleControl) ModuleControl {
	return ModuleControl{
		ID:              module.ID,
		Title:           module.Title,
		Description:     module.Description,
		Enabled:         module.Enabled,
		DispatchMode:    module.DispatchMode,
		ConsumerGroup:   module.ConsumerGroup,
		AllowedScopes:   cloneStrings(module.AllowedScopes),
		Tags:            cloneStrings(module.Tags),
		LatencyBudgetMS: module.LatencyBudgetMS,
	}
}

func clonePlugins(plugins []PluginControl) []PluginControl {
	if len(plugins) == 0 {
		return nil
	}
	out := make([]PluginControl, 0, len(plugins))
	for _, plugin := range plugins {
		out = append(out, clonePlugin(plugin))
	}
	return out
}

func clonePlugin(plugin PluginControl) PluginControl {
	return PluginControl{
		ID:           plugin.ID,
		Version:      plugin.Version,
		Runtime:      plugin.Runtime,
		Protocol:     plugin.Protocol,
		Status:       plugin.Status,
		Entry:        plugin.Entry,
		Description:  plugin.Description,
		Capabilities: cloneCapabilities(plugin.Capabilities),
	}
}

func cloneCapabilities(capabilities []PluginCapability) []PluginCapability {
	if len(capabilities) == 0 {
		return nil
	}
	out := make([]PluginCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, PluginCapability{
			Module:  capability.Module,
			Actions: cloneStrings(capability.Actions),
			Scopes:  cloneStrings(capability.Scopes),
		})
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
