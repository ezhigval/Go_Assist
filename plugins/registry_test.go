package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestResolvesRelativeProcessEntry(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "finance.plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{
		"id": "finance-sync",
		"version": "1.0.0",
		"display_name": "Finance Sync",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/finance-sync",
		"capabilities": [
			{"module": "finance", "actions": ["create_transaction", "sync"], "scopes": ["business", "work"]}
		]
	}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	expectedEntry := filepath.Join(dir, "bin", "finance-sync")
	if manifest.EntryPath != expectedEntry {
		t.Fatalf("EntryPath = %q, want %q", manifest.EntryPath, expectedEntry)
	}
	if !manifest.Supports("finance", "create_transaction") {
		t.Fatalf("manifest should support finance/create_transaction")
	}
	if manifest.Supports("tracker", "create_reminder") {
		t.Fatalf("manifest should not support tracker/create_reminder")
	}
}

func TestLoadManifestRejectsRelativeEscape(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "escape.plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{
		"id": "escape-plugin",
		"version": "1.0.0",
		"runtime": "process",
		"protocol": "stdio",
		"entry": "../outside",
		"capabilities": [
			{"module": "knowledge", "actions": ["save_note"]}
		]
	}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := LoadManifest(manifestPath)
	if err == nil {
		t.Fatal("LoadManifest returned nil error, want ErrManifestEntryEscapesDir")
	}
	if !errorsIs(err, ErrManifestEntryEscapesDir) {
		t.Fatalf("LoadManifest error = %v, want ErrManifestEntryEscapesDir", err)
	}
}

func TestLoadManifestRejectsAbsoluteEntryPath(t *testing.T) {
	dir := t.TempDir()
	absoluteEntry := filepath.Join(dir, "bin", "absolute-plugin")
	manifestPath := filepath.Join(dir, "absolute.plugin.json")
	if err := os.WriteFile(manifestPath, []byte(fmt.Sprintf(`{
		"id": "absolute-plugin",
		"version": "1.0.0",
		"runtime": "process",
		"protocol": "stdio",
		"entry": %q,
		"capabilities": [
			{"module": "knowledge", "actions": ["save_note"]}
		]
	}`, absoluteEntry)), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := LoadManifest(manifestPath)
	if err == nil {
		t.Fatal("LoadManifest returned nil error, want ErrManifestEntryAbsolute")
	}
	if !errorsIs(err, ErrManifestEntryAbsolute) {
		t.Fatalf("LoadManifest error = %v, want ErrManifestEntryAbsolute", err)
	}
}

func TestParseManifestRejectsUnsupportedPermission(t *testing.T) {
	_, err := ParseManifest([]byte(`{
		"id": "unsafe-plugin",
		"version": "1.0.0",
		"runtime": "process",
		"protocol": "grpc",
		"entry": "bin/unsafe-plugin",
		"permissions": ["root.exec"],
		"capabilities": [
			{"module": "finance", "actions": ["sync"]}
		]
	}`))
	if err == nil {
		t.Fatal("ParseManifest returned nil error, want unsupported permission")
	}
	if got := err.Error(); got != `plugins: unsupported permission "root.exec"` {
		t.Fatalf("ParseManifest error = %q, want unsupported permission", got)
	}
}

func TestParseManifestRejectsWASMGRPCProtocol(t *testing.T) {
	_, err := ParseManifest([]byte(`{
		"id": "tracker-plan",
		"version": "1.0.0",
		"runtime": "wasm",
		"protocol": "grpc",
		"entry": "tracker-plan.wasm",
		"capabilities": [
			{"module": "tracker", "actions": ["create_task"]}
		]
	}`))
	if err == nil {
		t.Fatal("ParseManifest returned nil error, want wasm protocol validation")
	}
	if got := err.Error(); got != `plugins: wasm manifest "tracker-plan" must use stdio protocol when protocol is set` {
		t.Fatalf("ParseManifest error = %q, want wasm protocol validation", got)
	}
}

func TestRegistryResolveReturnsMatchingVersionedPlugins(t *testing.T) {
	registry := NewRegistry()

	first := LoadedManifest{
		Manifest: Manifest{
			ID:       "finance-sync",
			Version:  "1.0.0",
			Runtime:  RuntimeProcess,
			Protocol: ProtocolGRPC,
			Entry:    "bin/finance-sync",
			Capabilities: []Capability{
				{Module: "finance", Actions: []string{"create_transaction"}},
			},
		},
		SourcePath: "/tmp/finance-sync.plugin.json",
		EntryPath:  "/tmp/bin/finance-sync",
	}
	second := LoadedManifest{
		Manifest: Manifest{
			ID:      "tracker-plan",
			Version: "1.1.0",
			Runtime: RuntimeWASM,
			Entry:   "tracker-plan.wasm",
			Capabilities: []Capability{
				{Module: "tracker", Actions: []string{"create_reminder", "create_task"}},
			},
		},
		SourcePath: "/tmp/tracker-plan.plugin.json",
		EntryPath:  "/tmp/tracker-plan.wasm",
	}

	if err := registry.Register(first); err != nil {
		t.Fatalf("Register(first) returned error: %v", err)
	}
	if err := registry.Register(second); err != nil {
		t.Fatalf("Register(second) returned error: %v", err)
	}

	matches := registry.Resolve("tracker", "create_task")
	if len(matches) != 1 {
		t.Fatalf("Resolve returned %d matches, want 1", len(matches))
	}
	if matches[0].ID != "tracker-plan" || matches[0].Version != "1.1.0" {
		t.Fatalf("unexpected resolve result: %+v", matches[0])
	}

	list := registry.List()
	if len(list) != 2 {
		t.Fatalf("List returned %d manifests, want 2", len(list))
	}
	if list[0].Key() != "finance-sync@1.0.0" || list[1].Key() != "tracker-plan@1.1.0" {
		t.Fatalf("unexpected registry order: %+v", list)
	}
}

func TestRegistryRegisterRejectsMismatchedEntryPath(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(LoadedManifest{
		Manifest: Manifest{
			ID:       "finance-sync",
			Version:  "1.0.0",
			Runtime:  RuntimeProcess,
			Protocol: ProtocolGRPC,
			Entry:    "bin/finance-sync",
			Capabilities: []Capability{
				{Module: "finance", Actions: []string{"create_transaction"}},
			},
		},
		SourcePath: "/tmp/finance-sync.plugin.json",
		EntryPath:  "/etc/finance-sync",
	})
	if err == nil {
		t.Fatal("Register returned nil error, want ErrManifestEntryPathMismatch")
	}
	if !errorsIs(err, ErrManifestEntryPathMismatch) {
		t.Fatalf("Register error = %v, want ErrManifestEntryPathMismatch", err)
	}
}

func errorsIs(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	return err == target || (err.Error() == target.Error())
}
