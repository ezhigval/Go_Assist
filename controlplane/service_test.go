package controlplane

import (
	"context"
	"os"
	"path/filepath"
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

func TestServiceHealthStatusReflectsPersistenceAndPluginHydration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	service, err := NewPersistentService(path)
	if err != nil {
		t.Fatalf("NewPersistentService returned error: %v", err)
	}

	dir := t.TempDir()
	writeManifest(t, dir, "finance-sync.plugin.json", `{
		"id": "finance-sync",
		"version": "1.0.0",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/finance-sync",
		"capabilities": [
			{"module": "finance", "actions": ["create_transaction"], "scopes": ["business"]}
		]
	}`)

	if err := service.HydratePluginsFromDir(context.Background(), dir); err != nil {
		t.Fatalf("HydratePluginsFromDir returned error: %v", err)
	}

	status, err := service.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !status.OK || !status.PersistEnabled {
		t.Fatalf("health status = %+v, want ok+persistent", status)
	}
	if status.Mode != "persistent" {
		t.Fatalf("health mode = %q, want persistent", status.Mode)
	}
	if status.PersistPath != path {
		t.Fatalf("health persist path = %q, want %q", status.PersistPath, path)
	}
	if status.PluginDir != dir || status.PluginManifests != 1 {
		t.Fatalf("health plugin fields = %+v, want dir=%q count=1", status, dir)
	}
	if status.UpdatedAt == "" {
		t.Fatal("health updated_at is empty")
	}
}

func TestPersistentServiceReloadsMutationsFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "controlplane.json")
	service, err := NewPersistentService(path)
	if err != nil {
		t.Fatalf("NewPersistentService returned error: %v", err)
	}

	ctx := context.Background()
	created, err := service.CreateScope(ctx, ScopePreset{
		Segment: "health",
		Tags:    []string{"focus"},
	})
	if err != nil {
		t.Fatalf("CreateScope returned error: %v", err)
	}

	enabled := false
	if _, err := service.UpdateModule(ctx, "tracker", ModulePatch{Enabled: &enabled}); err != nil {
		t.Fatalf("UpdateModule returned error: %v", err)
	}

	status := PluginStatusEnabled
	if _, err := service.UpdatePlugin(ctx, "audit-mirror", PluginPatch{Status: &status}); err != nil {
		t.Fatalf("UpdatePlugin returned error: %v", err)
	}

	if _, err := service.CycleBrokerMode(ctx, "runtime-core"); err != nil {
		t.Fatalf("CycleBrokerMode returned error: %v", err)
	}

	reloaded, err := NewPersistentService(path)
	if err != nil {
		t.Fatalf("NewPersistentService(reload) returned error: %v", err)
	}

	scopes, err := reloaded.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes returned error: %v", err)
	}
	foundScope := false
	for _, scope := range scopes {
		if ScopeKey(scope) == ScopeKey(created) {
			foundScope = true
			break
		}
	}
	if !foundScope {
		t.Fatalf("reloaded scopes missing %q", ScopeKey(created))
	}

	snapshot, err := reloaded.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}

	var tracker ModuleControl
	for _, module := range snapshot.Modules {
		if module.ID == "tracker" {
			tracker = module
			break
		}
	}
	if tracker.ID == "" || tracker.Enabled {
		t.Fatalf("reloaded tracker module = %+v, want disabled tracker", tracker)
	}

	var audit PluginControl
	for _, plugin := range snapshot.Plugins {
		if plugin.ID == "audit-mirror" {
			audit = plugin
			break
		}
	}
	if audit.ID == "" || audit.Status != PluginStatusEnabled {
		t.Fatalf("reloaded audit plugin = %+v, want enabled", audit)
	}

	var runtimeCore BrokerLane
	for _, broker := range snapshot.Brokers {
		if broker.ID == "runtime-core" {
			runtimeCore = broker
			break
		}
	}
	if runtimeCore.ID == "" || runtimeCore.Mode != BrokerModeNATS || runtimeCore.Status != BrokerStatusPlanned {
		t.Fatalf("reloaded runtime-core broker = %+v, want nats/planned", runtimeCore)
	}
}

func TestPersistentServiceRejectsBrokenSnapshotFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := NewPersistentService(path); err == nil {
		t.Fatal("NewPersistentService returned nil error for broken snapshot")
	}
}

func TestServiceHydratePluginsFromDirOverlaysRuntimeFieldsAndKeepsOperatorState(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "finance-sync.plugin.json", `{
		"id": "finance-sync",
		"version": "1.2.0",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/finance-sync-v2",
		"description": "manifest description",
		"capabilities": [
			{"module": "finance", "actions": ["create_transaction", "sync"], "scopes": ["business", "assets"]}
		]
	}`)
	writeManifest(t, dir, "ops-audit.plugin.json", `{
		"id": "ops-audit",
		"version": "0.1.0",
		"runtime": "process",
		"protocol": "stdio",
		"entry": "bin/ops-audit",
		"description": "new audit plugin",
		"capabilities": [
			{"module": "knowledge", "actions": ["save_note"], "scopes": ["business"]}
		]
	}`)

	service := NewService()
	ctx := context.Background()
	description := "operator override"
	if _, err := service.UpdatePlugin(ctx, "finance-sync", PluginPatch{Description: &description}); err != nil {
		t.Fatalf("UpdatePlugin returned error: %v", err)
	}

	if err := service.HydratePluginsFromDir(ctx, dir); err != nil {
		t.Fatalf("HydratePluginsFromDir returned error: %v", err)
	}

	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}

	var financeSync PluginControl
	var opsAudit PluginControl
	for _, plugin := range snapshot.Plugins {
		switch plugin.ID {
		case "finance-sync":
			financeSync = plugin
		case "ops-audit":
			opsAudit = plugin
		}
	}

	if financeSync.Version != "1.2.0" || financeSync.Entry != "bin/finance-sync-v2" {
		t.Fatalf("finance-sync runtime fields = %+v, want manifest overlay", financeSync)
	}
	if financeSync.Description != "operator override" {
		t.Fatalf("finance-sync description = %q, want operator override", financeSync.Description)
	}
	if financeSync.Status != PluginStatusEnabled {
		t.Fatalf("finance-sync status = %q, want enabled", financeSync.Status)
	}
	if len(financeSync.Capabilities) != 1 || len(financeSync.Capabilities[0].Scopes) != 2 || financeSync.Capabilities[0].Scopes[1] != "business" {
		t.Fatalf("finance-sync capabilities = %+v, want hydrated manifest scopes", financeSync.Capabilities)
	}

	if opsAudit.ID == "" {
		t.Fatal("ops-audit plugin was not appended from manifests")
	}
	if opsAudit.Status != PluginStatusStaged {
		t.Fatalf("ops-audit status = %q, want staged", opsAudit.Status)
	}
}

func TestServiceReloadPluginsUsesConfiguredManifestSource(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "finance-sync.plugin.json", `{
		"id": "finance-sync",
		"version": "1.0.0",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/finance-sync-v1",
		"capabilities": [
			{"module": "finance", "actions": ["create_transaction"], "scopes": ["business"]}
		]
	}`)

	service := NewService()
	ctx := context.Background()
	if err := service.HydratePluginsFromDir(ctx, dir); err != nil {
		t.Fatalf("HydratePluginsFromDir returned error: %v", err)
	}

	writeManifest(t, dir, "finance-sync.plugin.json", `{
		"id": "finance-sync",
		"version": "1.3.0",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/finance-sync-v2",
		"description": "reload check",
		"capabilities": [
			{"module": "finance", "actions": ["create_transaction", "reconcile"], "scopes": ["assets", "business"]}
		]
	}`)

	snapshot, err := service.ReloadPlugins(ctx)
	if err != nil {
		t.Fatalf("ReloadPlugins returned error: %v", err)
	}

	var financeSync PluginControl
	for _, plugin := range snapshot.Plugins {
		if plugin.ID == "finance-sync" {
			financeSync = plugin
			break
		}
	}
	if financeSync.Version != "1.3.0" || financeSync.Entry != "bin/finance-sync-v2" {
		t.Fatalf("finance-sync after reload = %+v, want refreshed runtime metadata", financeSync)
	}
	if len(financeSync.Capabilities) != 1 || len(financeSync.Capabilities[0].Actions) != 2 {
		t.Fatalf("finance-sync capabilities after reload = %+v, want manifest refresh", financeSync.Capabilities)
	}

	status, err := service.Health(ctx)
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if status.PluginDir != dir || status.PluginManifests != 1 {
		t.Fatalf("health after reload = %+v, want dir=%q count=1", status, dir)
	}
}

func writeManifest(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", name, err)
	}
}
