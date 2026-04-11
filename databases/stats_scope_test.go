package databases

import (
	"reflect"
	"testing"
)

func TestExtractStatsScopeUsesReservedKey(t *testing.T) {
	t.Parallel()

	scope, metadata := extractStatsScope(map[string]interface{}{
		"_scope": "business",
		"label":  "monthly_report",
	})

	if scope != "business" {
		t.Fatalf("scope = %q, want business", scope)
	}
	want := map[string]interface{}{"label": "monthly_report"}
	if !reflect.DeepEqual(metadata, want) {
		t.Fatalf("metadata = %+v, want %+v", metadata, want)
	}
}

func TestExtractStatsScopeFallsBackToScopeField(t *testing.T) {
	t.Parallel()

	scope, metadata := extractStatsScope(map[string]interface{}{
		"scope": "travel",
		"kind":  "sync",
	})

	if scope != "travel" {
		t.Fatalf("scope = %q, want travel", scope)
	}
	want := map[string]interface{}{"scope": "travel", "kind": "sync"}
	if !reflect.DeepEqual(metadata, want) {
		t.Fatalf("metadata = %+v, want %+v", metadata, want)
	}
}

func TestExtractStatsScopeDefaultsToPersonal(t *testing.T) {
	t.Parallel()

	scope, metadata := extractStatsScope(nil)
	if scope != "personal" {
		t.Fatalf("scope = %q, want personal", scope)
	}
	if metadata != nil {
		t.Fatalf("metadata = %+v, want nil", metadata)
	}
}
