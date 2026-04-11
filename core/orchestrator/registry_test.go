package orchestrator

import "testing"

func TestModuleRegistryRegisterAndLookup(t *testing.T) {
	reg := NewModuleRegistry()

	if err := reg.RegisterModule("finance", []string{"create_transaction", "set_budget"}); err != nil {
		t.Fatalf("RegisterModule returned error: %v", err)
	}

	if !reg.HasEndpoint("finance", "create_transaction") {
		t.Fatalf("expected registered endpoint to exist")
	}
	if reg.HasEndpoint("finance", "missing") {
		t.Fatalf("unexpected endpoint reported as existing")
	}
	if len(reg.ListModules()) != 1 {
		t.Fatalf("ListModules returned unexpected size: %d", len(reg.ListModules()))
	}
}

func TestModuleRegistryRejectsEmptyName(t *testing.T) {
	reg := NewModuleRegistry()
	if err := reg.RegisterModule("", []string{"create"}); err == nil {
		t.Fatalf("expected error for empty module name")
	}
}
