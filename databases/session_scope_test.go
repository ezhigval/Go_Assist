package databases

import (
	"reflect"
	"testing"
)

func TestExtractSessionActiveScopeStripsReservedPayloadKey(t *testing.T) {
	t.Parallel()

	scope, payload := extractSessionActiveScope(map[string]interface{}{
		"_active_scope": "business",
		"draft":         "note",
	})

	if scope != "business" {
		t.Fatalf("scope = %q, want business", scope)
	}
	want := map[string]interface{}{"draft": "note"}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("payload = %+v, want %+v", payload, want)
	}
}

func TestExtractSessionActiveScopeFallsBackToDefault(t *testing.T) {
	t.Parallel()

	scope, payload := extractSessionActiveScope(map[string]interface{}{
		"_active_scope": "unknown",
	})

	if scope != "personal" {
		t.Fatalf("scope = %q, want personal", scope)
	}
	if payload != nil {
		t.Fatalf("payload = %+v, want nil", payload)
	}
}

func TestHydrateSessionPayloadAddsReservedScopeKey(t *testing.T) {
	t.Parallel()

	payload := hydrateSessionPayload(map[string]interface{}{"draft": "note"}, "travel")
	want := map[string]interface{}{
		"_active_scope": "travel",
		"draft":         "note",
	}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("payload = %+v, want %+v", payload, want)
	}
}
