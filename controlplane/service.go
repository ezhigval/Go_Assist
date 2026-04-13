package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"modulr/events"
	"modulr/plugins"
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
	mu          sync.RWMutex
	snapshot    Snapshot
	persistPath string
	pluginDir   string
	pluginCount int
}

// NewService создаёт control-plane projection со snapshot по умолчанию.
func NewService() *Service {
	return &Service{
		snapshot: defaultSnapshot(),
	}
}

// NewPersistentService создаёт control-plane projection с file-backed snapshot.
func NewPersistentService(path string) (*Service, error) {
	if path == "" {
		return NewService(), nil
	}

	snapshot, err := loadSnapshotFromFile(path)
	if err != nil {
		return nil, err
	}

	return &Service{
		snapshot:    snapshot,
		persistPath: path,
	}, nil
}

// HydratePluginsFromDir подтягивает plugin projection из реальных manifests.
func (s *Service) HydratePluginsFromDir(ctx context.Context, dir string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}

	loaded, err := plugins.LoadDir(dir)
	if err != nil {
		return fmt.Errorf("controlplane: hydrate plugins from %q: %w", dir, err)
	}
	if len(loaded) == 0 {
		s.mu.Lock()
		s.pluginDir = dir
		s.pluginCount = 0
		s.mu.Unlock()
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneSnapshot(s.snapshot)
	next.Plugins = mergePluginsWithManifests(next.Plugins, loaded)
	if err := s.commitLocked(next); err != nil {
		return err
	}
	s.pluginDir = dir
	s.pluginCount = len(loaded)
	return nil
}

// Health подтверждает готовность in-memory projection.
func (s *Service) Health(ctx context.Context) (HealthStatus, error) {
	if ctx.Err() != nil {
		return HealthStatus{}, ctx.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	mode := "memory"
	if s.persistPath != "" {
		mode = "persistent"
	}

	return HealthStatus{
		OK:              true,
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
		Mode:            mode,
		PersistEnabled:  s.persistPath != "",
		PersistPath:     s.persistPath,
		PluginDir:       s.pluginDir,
		PluginManifests: s.pluginCount,
		UpdatedAt:       s.snapshot.UpdatedAt,
	}, nil
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

	next := cloneSnapshot(s.snapshot)
	id := ScopeKey(scope)
	if existing, index := findScopeLocked(next.Scopes, id); index >= 0 {
		return cloneScope(existing), nil
	}

	next.Scopes = append(next.Scopes, scope)
	if err := s.commitLocked(next); err != nil {
		return ScopePreset{}, err
	}
	return cloneScope(scope), nil
}

// UpdateScopeTags обновляет tags у существующего preset.
func (s *Service) UpdateScopeTags(ctx context.Context, id string, tags []string) (ScopePreset, error) {
	if ctx.Err() != nil {
		return ScopePreset{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneSnapshot(s.snapshot)
	scope, index := findScopeLocked(next.Scopes, id)
	if index < 0 {
		return ScopePreset{}, ErrScopeNotFound
	}

	updated := scope
	updated.Tags = normalizeTags(tags)
	updatedID := ScopeKey(updated)
	if updatedID != id {
		if _, conflictIndex := findScopeLocked(next.Scopes, updatedID); conflictIndex >= 0 {
			return ScopePreset{}, ErrScopeConflict
		}
	}

	next.Scopes[index] = updated
	if err := s.commitLocked(next); err != nil {
		return ScopePreset{}, err
	}
	return cloneScope(updated), nil
}

// DeleteScope удаляет preset, но не позволяет оставить систему без scope.
func (s *Service) DeleteScope(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneSnapshot(s.snapshot)
	_, index := findScopeLocked(next.Scopes, id)
	if index < 0 {
		return ErrScopeNotFound
	}
	if len(next.Scopes) <= 1 {
		return ErrLastScope
	}

	next.Scopes = append(next.Scopes[:index], next.Scopes[index+1:]...)
	return s.commitLocked(next)
}

// UpdateModule обновляет operator-facing module-control.
func (s *Service) UpdateModule(ctx context.Context, id string, patch ModulePatch) (ModuleControl, error) {
	if ctx.Err() != nil {
		return ModuleControl{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneSnapshot(s.snapshot)
	for i, module := range next.Modules {
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
		next.Modules[i] = module
		if err := s.commitLocked(next); err != nil {
			return ModuleControl{}, err
		}
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

	next := cloneSnapshot(s.snapshot)
	for i, plugin := range next.Plugins {
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
		next.Plugins[i] = plugin
		if err := s.commitLocked(next); err != nil {
			return PluginControl{}, err
		}
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

	next := cloneSnapshot(s.snapshot)
	for i, broker := range next.Brokers {
		if broker.ID != id {
			continue
		}
		broker.Mode, broker.Status = rotateBrokerMode(broker.Mode)
		next.Brokers[i] = broker
		if err := s.commitLocked(next); err != nil {
			return BrokerLane{}, err
		}
		return cloneBroker(broker), nil
	}

	return BrokerLane{}, ErrBrokerNotFound
}

func (s *Service) commitLocked(snapshot Snapshot) error {
	snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if s.persistPath != "" {
		if err := saveSnapshotToFile(s.persistPath, snapshot); err != nil {
			return err
		}
	}
	s.snapshot = snapshot
	return nil
}

func loadSnapshotFromFile(path string) (Snapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultSnapshot(), nil
		}
		return Snapshot{}, fmt.Errorf("controlplane: read snapshot %q: %w", path, err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("controlplane: decode snapshot %q: %w", path, err)
	}
	if err := normalizeSnapshotSeed(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("controlplane: validate snapshot %q: %w", path, err)
	}
	if snapshot.UpdatedAt == "" {
		snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return snapshot, nil
}

func saveSnapshotToFile(path string, snapshot Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("controlplane: create state dir for %q: %w", path, err)
	}

	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("controlplane: encode snapshot %q: %w", path, err)
	}
	payload = append(payload, '\n')

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "controlplane-*.json")
	if err != nil {
		return fmt.Errorf("controlplane: create temp snapshot for %q: %w", path, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("controlplane: write temp snapshot for %q: %w", path, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("controlplane: close temp snapshot for %q: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("controlplane: replace snapshot %q: %w", path, err)
	}
	return nil
}

func mergePluginsWithManifests(current []PluginControl, manifests []plugins.LoadedManifest) []PluginControl {
	if len(manifests) == 0 {
		return clonePlugins(current)
	}

	out := clonePlugins(current)
	indexByID := make(map[string]int, len(out))
	for i, plugin := range out {
		indexByID[plugin.ID] = i
	}

	latestByID := make(map[string]plugins.LoadedManifest, len(manifests))
	for _, manifest := range manifests {
		existing, ok := latestByID[manifest.ID]
		if !ok || compareManifestVersion(manifest.Version, existing.Version) > 0 {
			latestByID[manifest.ID] = manifest
		}
	}

	ids := make([]string, 0, len(latestByID))
	for id := range latestByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		projected := pluginControlFromManifest(latestByID[id])
		if index, ok := indexByID[id]; ok {
			existing := out[index]
			projected.Status = existing.Status
			if existing.Description != "" {
				projected.Description = existing.Description
			}
			out[index] = projected
			continue
		}
		out = append(out, projected)
	}

	return out
}

func pluginControlFromManifest(manifest plugins.LoadedManifest) PluginControl {
	capabilities := make([]PluginCapability, 0, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		capabilities = append(capabilities, PluginCapability{
			Module:  capability.Module,
			Actions: cloneStrings(capability.Actions),
			Scopes:  cloneStrings(capability.Scopes),
		})
	}

	protocol := PluginProtocol(manifest.Protocol)
	if protocol == "" {
		protocol = PluginProtocolStdio
	}

	return PluginControl{
		ID:           manifest.ID,
		Version:      manifest.Version,
		Runtime:      PluginRuntime(manifest.Runtime),
		Protocol:     protocol,
		Status:       PluginStatusStaged,
		Entry:        manifest.Entry,
		Description:  manifest.Description,
		Capabilities: capabilities,
	}
}

func compareManifestVersion(left, right string) int {
	leftMajor, leftMinor, leftPatch := parseManifestVersion(left)
	rightMajor, rightMinor, rightPatch := parseManifestVersion(right)

	switch {
	case leftMajor != rightMajor:
		return compareInt(leftMajor, rightMajor)
	case leftMinor != rightMinor:
		return compareInt(leftMinor, rightMinor)
	case leftPatch != rightPatch:
		return compareInt(leftPatch, rightPatch)
	default:
		return strings.Compare(left, right)
	}
}

func parseManifestVersion(value string) (int, int, int) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if cut := strings.IndexAny(value, "-+"); cut >= 0 {
		value = value[:cut]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return 0, 0, 0
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0
	}
	return major, minor, patch
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func normalizeScope(scope ScopePreset) (ScopePreset, error) {
	scope.Segment = string(events.ParseSegmentFromAny(scope.Segment))
	if !validScope(scope.Segment) {
		return ScopePreset{}, ErrInvalidScope
	}
	scope.Tags = normalizeTags(scope.Tags)
	hadMetadata := scope.Metadata != nil
	scope.Metadata = cloneAnyMap(scope.Metadata)
	if scope.Metadata == nil && hadMetadata {
		scope.Metadata = map[string]any{}
	}
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
