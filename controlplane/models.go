package controlplane

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"modulr/events"
)

// BrokerMode описывает backend-режим transport lane.
type BrokerMode string

const (
	BrokerModeMemory BrokerMode = "memory"
	BrokerModeNATS   BrokerMode = "nats"
	BrokerModeKafka  BrokerMode = "kafka"
)

// BrokerStatus отражает readiness lane.
type BrokerStatus string

const (
	BrokerStatusReady    BrokerStatus = "ready"
	BrokerStatusPlanned  BrokerStatus = "planned"
	BrokerStatusDegraded BrokerStatus = "degraded"
)

// AckPolicy описывает delivery semantics consumer group.
type AckPolicy string

const (
	AckPolicyAtLeastOnce AckPolicy = "at_least_once"
	AckPolicyExactlyOnce AckPolicy = "exactly_once"
)

// ModuleDispatchMode описывает shape dispatch path.
type ModuleDispatchMode string

const (
	ModuleDispatchInline ModuleDispatchMode = "inline"
	ModuleDispatchQueued ModuleDispatchMode = "queued"
	ModuleDispatchFanout ModuleDispatchMode = "fanout"
)

// PluginRuntime описывает способ исполнения плагина.
type PluginRuntime string

const (
	PluginRuntimeProcess PluginRuntime = "process"
	PluginRuntimeWASM    PluginRuntime = "wasm"
)

// PluginProtocol описывает host/plugin wire protocol.
type PluginProtocol string

const (
	PluginProtocolGRPC  PluginProtocol = "grpc"
	PluginProtocolStdio PluginProtocol = "stdio"
)

// PluginStatus описывает rollout state плагина.
type PluginStatus string

const (
	PluginStatusEnabled  PluginStatus = "enabled"
	PluginStatusStaged   PluginStatus = "staged"
	PluginStatusDisabled PluginStatus = "disabled"
)

// ScopePreset — operator-facing представление life scope.
type ScopePreset struct {
	Segment  string         `json:"segment"`
	Tags     []string       `json:"tags"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// BrokerConsumerGroup описывает consumer-group форму lane.
type BrokerConsumerGroup struct {
	ID        string    `json:"id"`
	Consumers int       `json:"consumers"`
	Lag       int       `json:"lag"`
	AckPolicy AckPolicy `json:"ackPolicy"`
}

// BrokerLane — операторский snapshot distributed lane.
type BrokerLane struct {
	ID             string                `json:"id"`
	Title          string                `json:"title"`
	Topic          string                `json:"topic"`
	Mode           BrokerMode            `json:"mode"`
	Status         BrokerStatus          `json:"status"`
	Notes          string                `json:"notes"`
	ConsumerGroups []BrokerConsumerGroup `json:"consumerGroups"`
}

// ModuleControl — control-plane snapshot доменного admission path.
type ModuleControl struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	Enabled         bool               `json:"enabled"`
	DispatchMode    ModuleDispatchMode `json:"dispatchMode"`
	ConsumerGroup   string             `json:"consumerGroup"`
	AllowedScopes   []string           `json:"allowedScopes"`
	Tags            []string           `json:"tags"`
	LatencyBudgetMS int                `json:"latencyBudgetMs"`
}

// PluginCapability — module/action coverage плагина.
type PluginCapability struct {
	Module  string   `json:"module"`
	Actions []string `json:"actions"`
	Scopes  []string `json:"scopes"`
}

// PluginControl — operator-facing projection plugin manifest'а.
type PluginControl struct {
	ID           string             `json:"id"`
	Version      string             `json:"version"`
	Runtime      PluginRuntime      `json:"runtime"`
	Protocol     PluginProtocol     `json:"protocol"`
	Status       PluginStatus       `json:"status"`
	Entry        string             `json:"entry"`
	Description  string             `json:"description"`
	Capabilities []PluginCapability `json:"capabilities"`
}

// Snapshot — полный control-plane snapshot для frontend.
type Snapshot struct {
	UpdatedAt  string          `json:"updatedAt"`
	Scopes     []ScopePreset   `json:"scopes"`
	TagPresets []string        `json:"tagPresets"`
	Brokers    []BrokerLane    `json:"brokers"`
	Modules    []ModuleControl `json:"modules"`
	Plugins    []PluginControl `json:"plugins"`
}

// ModulePatch допускает частичное обновление module-control.
type ModulePatch struct {
	Enabled         *bool               `json:"enabled,omitempty"`
	DispatchMode    *ModuleDispatchMode `json:"dispatchMode,omitempty"`
	ConsumerGroup   *string             `json:"consumerGroup,omitempty"`
	AllowedScopes   *[]string           `json:"allowedScopes,omitempty"`
	Tags            *[]string           `json:"tags,omitempty"`
	LatencyBudgetMS *int                `json:"latencyBudgetMs,omitempty"`
}

// PluginPatch допускает частичное обновление plugin-control.
type PluginPatch struct {
	Status       *PluginStatus       `json:"status,omitempty"`
	Description  *string             `json:"description,omitempty"`
	Capabilities *[]PluginCapability `json:"capabilities,omitempty"`
}

var (
	//go:embed default_snapshot.json
	defaultSnapshotRaw []byte

	defaultSnapshotSeedOnce sync.Once
	defaultSnapshotSeed     Snapshot
	defaultSnapshotSeedErr  error
)

func (m BrokerMode) valid() bool {
	switch m {
	case BrokerModeMemory, BrokerModeNATS, BrokerModeKafka:
		return true
	default:
		return false
	}
}

func (s BrokerStatus) valid() bool {
	switch s {
	case BrokerStatusReady, BrokerStatusPlanned, BrokerStatusDegraded:
		return true
	default:
		return false
	}
}

func (m ModuleDispatchMode) valid() bool {
	switch m {
	case ModuleDispatchInline, ModuleDispatchQueued, ModuleDispatchFanout:
		return true
	default:
		return false
	}
}

func (s PluginStatus) valid() bool {
	switch s {
	case PluginStatusEnabled, PluginStatusStaged, PluginStatusDisabled:
		return true
	default:
		return false
	}
}

func rotateBrokerMode(mode BrokerMode) (BrokerMode, BrokerStatus) {
	switch mode {
	case BrokerModeMemory:
		return BrokerModeNATS, BrokerStatusPlanned
	case BrokerModeNATS:
		return BrokerModeKafka, BrokerStatusDegraded
	case BrokerModeKafka:
		fallthrough
	default:
		return BrokerModeMemory, BrokerStatusReady
	}
}

// ScopeKey повторяет frontend-идентификатор scope preset.
func ScopeKey(scope ScopePreset) string {
	return scope.Segment + ":" + strings.Join(normalizeTags(scope.Tags), ",")
}

func validScope(segment string) bool {
	return events.IsValidSegment(events.Segment(segment))
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		set[tag] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for tag := range set {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func defaultSnapshot() Snapshot {
	snapshot := cloneSnapshot(mustDefaultSnapshotSeed())
	snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return snapshot
}

func mustDefaultSnapshotSeed() Snapshot {
	defaultSnapshotSeedOnce.Do(func() {
		defaultSnapshotSeed, defaultSnapshotSeedErr = loadDefaultSnapshotSeed()
	})
	if defaultSnapshotSeedErr != nil {
		panic(defaultSnapshotSeedErr)
	}
	return cloneSnapshot(defaultSnapshotSeed)
}

func loadDefaultSnapshotSeed() (Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(defaultSnapshotRaw, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("controlplane: decode default snapshot seed: %w", err)
	}
	if err := normalizeSnapshotSeed(&snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func normalizeSnapshotSeed(snapshot *Snapshot) error {
	normalizedScopes := make([]ScopePreset, 0, len(snapshot.Scopes))
	for _, scope := range snapshot.Scopes {
		normalized, err := normalizeScope(scope)
		if err != nil {
			return err
		}
		normalizedScopes = append(normalizedScopes, normalized)
	}
	snapshot.Scopes = normalizedScopes
	snapshot.TagPresets = normalizeTags(snapshot.TagPresets)

	for i, broker := range snapshot.Brokers {
		if broker.ID == "" || broker.Topic == "" {
			return fmt.Errorf("controlplane: broker seed requires id/topic")
		}
		if !broker.Mode.valid() {
			return fmt.Errorf("controlplane: invalid broker mode %q", broker.Mode)
		}
		if !broker.Status.valid() {
			return fmt.Errorf("controlplane: invalid broker status %q", broker.Status)
		}
		snapshot.Brokers[i].ConsumerGroups = cloneConsumerGroups(broker.ConsumerGroups)
	}

	for i, module := range snapshot.Modules {
		if module.ID == "" {
			return fmt.Errorf("controlplane: module seed requires id")
		}
		if !module.DispatchMode.valid() {
			return fmt.Errorf("controlplane: invalid module dispatch mode %q", module.DispatchMode)
		}
		if !validScopeList(module.AllowedScopes) {
			return ErrInvalidScope
		}
		snapshot.Modules[i].AllowedScopes = cloneStrings(module.AllowedScopes)
		snapshot.Modules[i].Tags = normalizeTags(module.Tags)
	}

	for i, plugin := range snapshot.Plugins {
		if plugin.ID == "" {
			return fmt.Errorf("controlplane: plugin seed requires id")
		}
		if !plugin.Status.valid() {
			return fmt.Errorf("controlplane: invalid plugin status %q", plugin.Status)
		}
		if !validCapabilities(plugin.Capabilities) {
			return ErrInvalidScope
		}
		snapshot.Plugins[i].Capabilities = cloneCapabilities(plugin.Capabilities)
	}

	return nil
}
