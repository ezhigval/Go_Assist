package controlplane

import "context"

// API задаёт backend-контракт для operator-facing control plane.
type API interface {
	Health(ctx context.Context) error
	Snapshot(ctx context.Context) (Snapshot, error)
	ListScopes(ctx context.Context) ([]ScopePreset, error)
	CreateScope(ctx context.Context, scope ScopePreset) (ScopePreset, error)
	UpdateScopeTags(ctx context.Context, id string, tags []string) (ScopePreset, error)
	DeleteScope(ctx context.Context, id string) error
	UpdateModule(ctx context.Context, id string, patch ModulePatch) (ModuleControl, error)
	UpdatePlugin(ctx context.Context, id string, patch PluginPatch) (PluginControl, error)
	CycleBrokerMode(ctx context.Context, id string) (BrokerLane, error)
}
