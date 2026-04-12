package events

import "testing"

func TestNormalizeRoles(t *testing.T) {
	t.Parallel()

	got := NormalizeRoles([]any{"Admin", "user", "admin", "", "guest"})
	want := []string{"admin", "guest", "user"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeRoles() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeRoles() = %v, want %v", got, want)
		}
	}
}

func TestRolesAllowEvent(t *testing.T) {
	t.Parallel()

	if !RolesAllowEvent([]string{"guest"}, string(V1SystemStartup)) {
		t.Fatal("expected guest to allow v1.system.startup")
	}
	if RolesAllowEvent([]string{"guest"}, "v1.finance.create_transaction") {
		t.Fatal("expected guest to reject finance event")
	}
	if !RolesAllowEvent([]string{"user"}, "v1.finance.create_transaction") {
		t.Fatal("expected user to allow finance event")
	}
}

func TestMetadataAuthRequired(t *testing.T) {
	t.Parallel()

	if !MetadataAuthRequired(map[string]any{"auth_required": true}) {
		t.Fatal("expected bool true to require auth")
	}
	if !MetadataAuthRequired(map[string]any{"auth_required": "true"}) {
		t.Fatal("expected string true to require auth")
	}
	if MetadataAuthRequired(nil) {
		t.Fatal("expected nil metadata to not require auth")
	}
}
