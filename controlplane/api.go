package controlplane

import "context"

// HealthStatus — операторский статус control plane backend.
type HealthStatus struct {
	OK              bool   `json:"ok"`
	CheckedAt       string `json:"checked_at"`
	Mode            string `json:"mode"`
	PersistEnabled  bool   `json:"persist_enabled"`
	PersistPath     string `json:"persist_path,omitempty"`
	PluginDir       string `json:"plugin_dir,omitempty"`
	PluginManifests int    `json:"plugin_manifests"`
	UpdatedAt       string `json:"snapshot_updated_at"`
}

// API задаёт backend-контракт для operator-facing control plane.
type API interface {
	Health(ctx context.Context) (HealthStatus, error)
	Snapshot(ctx context.Context) (Snapshot, error)
	ListScopes(ctx context.Context) ([]ScopePreset, error)
	CreateScope(ctx context.Context, scope ScopePreset) (ScopePreset, error)
	UpdateScopeTags(ctx context.Context, id string, tags []string) (ScopePreset, error)
	DeleteScope(ctx context.Context, id string) error
	UpdateModule(ctx context.Context, id string, patch ModulePatch) (ModuleControl, error)
	UpdatePlugin(ctx context.Context, id string, patch PluginPatch) (PluginControl, error)
	CycleBrokerMode(ctx context.Context, id string) (BrokerLane, error)
}
