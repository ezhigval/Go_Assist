package controlplane

import (
	"sort"
	"strings"
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
	return Snapshot{
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		Scopes:     defaultScopes(),
		TagPresets: []string{"ops", "focus", "handoff", "priority", "audit", "automation"},
		Brokers: []BrokerLane{
			{
				ID:     "runtime-core",
				Title:  "Runtime Core Bus",
				Topic:  "runtime.events",
				Mode:   BrokerModeMemory,
				Status: BrokerStatusReady,
				Notes:  "Single-process baseline for orchestrator + metrics + transport responses.",
				ConsumerGroups: []BrokerConsumerGroup{
					{ID: "orchestrator", Consumers: 2, Lag: 0, AckPolicy: AckPolicyAtLeastOnce},
					{ID: "metrics", Consumers: 1, Lag: 0, AckPolicy: AckPolicyAtLeastOnce},
				},
			},
			{
				ID:     "plugin-fanout",
				Title:  "Plugin Fanout Lane",
				Topic:  "plugins.dispatch",
				Mode:   BrokerModeNATS,
				Status: BrokerStatusPlanned,
				Notes:  "Reserved lane for v2 plugin workloads and backpressure separation.",
				ConsumerGroups: []BrokerConsumerGroup{
					{ID: "plugin-workers", Consumers: 3, Lag: 4, AckPolicy: AckPolicyAtLeastOnce},
				},
			},
		},
		Modules: []ModuleControl{
			{
				ID:              "tracker",
				Title:           "Tracker",
				Description:     "Напоминания, планы и milestone flow для scoped productivity.",
				Enabled:         true,
				DispatchMode:    ModuleDispatchQueued,
				ConsumerGroup:   "tracker-workers",
				AllowedScopes:   []string{"personal", "work", "business"},
				Tags:            []string{"reminders", "milestones"},
				LatencyBudgetMS: 250,
			},
			{
				ID:              "finance",
				Title:           "Finance",
				Description:     "Транзакции и журнал расходов с отдельной очередью для business scope.",
				Enabled:         true,
				DispatchMode:    ModuleDispatchFanout,
				ConsumerGroup:   "finance-ledger",
				AllowedScopes:   []string{"personal", "business", "assets"},
				Tags:            []string{"ledger", "vat", "budget"},
				LatencyBudgetMS: 180,
			},
			{
				ID:              "knowledge",
				Title:           "Knowledge",
				Description:     "Ноты и query capture с мягким fallback в локальное хранилище.",
				Enabled:         true,
				DispatchMode:    ModuleDispatchInline,
				ConsumerGroup:   "knowledge-cache",
				AllowedScopes:   []string{"personal", "work", "travel"},
				Tags:            []string{"notes", "search"},
				LatencyBudgetMS: 120,
			},
			{
				ID:              "notifications",
				Title:           "Notifications",
				Description:     "Delivery path для outcome/fallback и cross-platform уведомлений.",
				Enabled:         false,
				DispatchMode:    ModuleDispatchFanout,
				ConsumerGroup:   "notify-broadcast",
				AllowedScopes:   []string{"personal", "family", "work", "business"},
				Tags:            []string{"push", "transport"},
				LatencyBudgetMS: 90,
			},
		},
		Plugins: []PluginControl{
			{
				ID:          "finance-sync",
				Version:     "1.0.0",
				Runtime:     PluginRuntimeProcess,
				Protocol:    PluginProtocolGRPC,
				Status:      PluginStatusEnabled,
				Entry:       "plugins/finance-sync/bin/finance-sync",
				Description: "External ledger adapter for business finance dispatch.",
				Capabilities: []PluginCapability{
					{Module: "finance", Actions: []string{"create_transaction", "sync"}, Scopes: []string{"business", "work"}},
				},
			},
			{
				ID:          "tracker-plan",
				Version:     "1.1.0",
				Runtime:     PluginRuntimeWASM,
				Protocol:    PluginProtocolStdio,
				Status:      PluginStatusStaged,
				Entry:       "plugins/tracker-plan/tracker-plan.wasm",
				Description: "Planned sandbox plugin for decomposition and milestone shaping.",
				Capabilities: []PluginCapability{
					{Module: "tracker", Actions: []string{"create_task", "create_reminder"}, Scopes: []string{"personal", "work"}},
				},
			},
			{
				ID:          "audit-mirror",
				Version:     "0.9.0",
				Runtime:     PluginRuntimeProcess,
				Protocol:    PluginProtocolStdio,
				Status:      PluginStatusDisabled,
				Entry:       "plugins/audit-mirror/bin/audit-mirror",
				Description: "Mirror consumer for regulated audit trails before broker rollout.",
				Capabilities: []PluginCapability{
					{Module: "knowledge", Actions: []string{"save_note"}, Scopes: []string{"business"}},
				},
			},
		},
	}
}

func defaultScopes() []ScopePreset {
	return []ScopePreset{
		{Segment: string(events.SegmentPersonal), Tags: nil, Metadata: map[string]any{}},
		{Segment: string(events.SegmentFamily), Tags: nil, Metadata: map[string]any{}},
		{Segment: string(events.SegmentWork), Tags: nil, Metadata: map[string]any{}},
		{Segment: string(events.SegmentBusiness), Tags: nil, Metadata: map[string]any{}},
		{Segment: string(events.SegmentBusiness), Tags: []string{"ops"}, Metadata: map[string]any{"source": "v2-control-plane"}},
		{Segment: string(events.SegmentTravel), Tags: []string{"handoff"}, Metadata: map[string]any{"source": "v2-control-plane"}},
	}
}
