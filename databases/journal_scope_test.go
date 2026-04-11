package databases

import (
	"reflect"
	"testing"
)

func TestNewJournalScopeFilterIncludesBaseScopeOnlyByDefault(t *testing.T) {
	t.Parallel()

	filter, err := NewJournalScopeFilter("personal", nil, nil)
	if err != nil {
		t.Fatalf("NewJournalScopeFilter returned error: %v", err)
	}

	if filter.Unrestricted() {
		t.Fatal("expected restricted filter")
	}

	want := []string{"personal"}
	if got := filter.Scopes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Scopes() = %v, want %v", got, want)
	}
}

func TestNewJournalScopeFilterAllowsMetadataCrossScope(t *testing.T) {
	t.Parallel()

	filter, err := NewJournalScopeFilter("personal", nil, map[string]any{
		"allowed_scopes": []string{"business", "travel"},
	})
	if err != nil {
		t.Fatalf("NewJournalScopeFilter returned error: %v", err)
	}

	want := []string{"personal", "business", "travel"}
	if got := filter.Scopes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Scopes() = %v, want %v", got, want)
	}
	if !filter.Allows("business") || !filter.Allows("travel") {
		t.Fatalf("expected filter to allow explicit cross-scope reads: %+v", filter)
	}
}

func TestNewJournalScopeFilterAllowsTagCrossScope(t *testing.T) {
	t.Parallel()

	filter, err := NewJournalScopeFilter("family", []string{"allow_scope:health"}, nil)
	if err != nil {
		t.Fatalf("NewJournalScopeFilter returned error: %v", err)
	}
	if !filter.Allows("family") || !filter.Allows("health") {
		t.Fatalf("expected tag-driven scope access, got %+v", filter)
	}
}

func TestNewJournalScopeFilterRejectsInvalidBaseScope(t *testing.T) {
	t.Parallel()

	if _, err := NewJournalScopeFilter("unknown", nil, nil); err == nil {
		t.Fatal("expected invalid base scope error")
	}
}

func TestJournalScopeFilterValidateRejectsZeroValue(t *testing.T) {
	t.Parallel()

	if err := (JournalScopeFilter{}).validate(); err == nil {
		t.Fatal("expected zero-value filter validation error")
	}
}

func TestFullJournalScopeFilterBypassesScopeChecks(t *testing.T) {
	t.Parallel()

	filter := FullJournalScopeFilter()
	if err := filter.validate(); err != nil {
		t.Fatalf("validate() returned error: %v", err)
	}
	if scopes := filter.Scopes(); scopes != nil {
		t.Fatalf("Scopes() = %v, want nil for unrestricted access", scopes)
	}
	if !filter.Allows("business") {
		t.Fatal("expected unrestricted filter to allow any valid scope")
	}
}
