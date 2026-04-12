package controlplane

import (
	"context"
	"testing"
)

func TestServiceReturnsDefaultSnapshot(t *testing.T) {
	service := NewService()

	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(snapshot.Brokers) == 0 || len(snapshot.Modules) == 0 || len(snapshot.Plugins) == 0 {
		t.Fatalf("snapshot missing operator data: %+v", snapshot)
	}
	if len(snapshot.Scopes) < 4 {
		t.Fatalf("snapshot scopes = %d, want >= 4", len(snapshot.Scopes))
	}
}

func TestServiceScopeCRUDAndGuards(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	created, err := service.CreateScope(ctx, ScopePreset{
		Segment: "travel",
		Tags:    []string{" Ops ", "audit", "ops"},
	})
	if err != nil {
		t.Fatalf("CreateScope returned error: %v", err)
	}
	if key := ScopeKey(created); key != "travel:audit,ops" {
		t.Fatalf("ScopeKey(created) = %q, want travel:audit,ops", key)
	}

	updated, err := service.UpdateScopeTags(ctx, ScopeKey(created), []string{"handoff", "ops"})
	if err != nil {
		t.Fatalf("UpdateScopeTags returned error: %v", err)
	}
	if key := ScopeKey(updated); key != "travel:handoff,ops" {
		t.Fatalf("ScopeKey(updated) = %q, want travel:handoff,ops", key)
	}

	if err := service.DeleteScope(ctx, ScopeKey(updated)); err != nil {
		t.Fatalf("DeleteScope returned error: %v", err)
	}

	guard := NewService()
	scopes, err := guard.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes returned error: %v", err)
	}
	for len(scopes) > 1 {
		if err := guard.DeleteScope(ctx, ScopeKey(scopes[0])); err != nil {
			t.Fatalf("DeleteScope shrink returned error: %v", err)
		}
		scopes, err = guard.ListScopes(ctx)
		if err != nil {
			t.Fatalf("ListScopes returned error: %v", err)
		}
	}
	if err := guard.DeleteScope(ctx, ScopeKey(scopes[0])); err == nil {
		t.Fatal("DeleteScope returned nil error, want ErrLastScope")
	}
}

func TestServiceCycleBrokerAndUpdateModulePlugin(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	broker, err := service.CycleBrokerMode(ctx, "runtime-core")
	if err != nil {
		t.Fatalf("CycleBrokerMode returned error: %v", err)
	}
	if broker.Mode != BrokerModeNATS || broker.Status != BrokerStatusPlanned {
		t.Fatalf("unexpected broker after cycle: %+v", broker)
	}

	enabled := false
	module, err := service.UpdateModule(ctx, "tracker", ModulePatch{Enabled: &enabled})
	if err != nil {
		t.Fatalf("UpdateModule returned error: %v", err)
	}
	if module.Enabled {
		t.Fatalf("module enabled = %v, want false", module.Enabled)
	}

	status := PluginStatusEnabled
	plugin, err := service.UpdatePlugin(ctx, "audit-mirror", PluginPatch{Status: &status})
	if err != nil {
		t.Fatalf("UpdatePlugin returned error: %v", err)
	}
	if plugin.Status != PluginStatusEnabled {
		t.Fatalf("plugin status = %q, want enabled", plugin.Status)
	}
}
