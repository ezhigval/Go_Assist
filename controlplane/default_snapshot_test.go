package controlplane

import "testing"

func TestDefaultSnapshotSeedLoadsFromEmbeddedJSON(t *testing.T) {
	snapshot := defaultSnapshot()

	if snapshot.UpdatedAt == "" {
		t.Fatal("defaultSnapshot UpdatedAt is empty")
	}
	if len(snapshot.Scopes) != 6 {
		t.Fatalf("defaultSnapshot scopes = %d, want 6", len(snapshot.Scopes))
	}
	if metadata := snapshot.Scopes[0].Metadata; len(metadata) != 0 {
		t.Fatalf("defaultSnapshot first scope metadata = %+v, want empty map", metadata)
	}
	if source := snapshot.Scopes[4].Metadata["source"]; source != "v2-control-plane" {
		t.Fatalf("defaultSnapshot business ops source = %v, want v2-control-plane", source)
	}
	if len(snapshot.Brokers) != 2 {
		t.Fatalf("defaultSnapshot brokers = %d, want 2", len(snapshot.Brokers))
	}
	if len(snapshot.Modules) != 4 {
		t.Fatalf("defaultSnapshot modules = %d, want 4", len(snapshot.Modules))
	}
	if len(snapshot.Plugins) != 3 {
		t.Fatalf("defaultSnapshot plugins = %d, want 3", len(snapshot.Plugins))
	}
}
