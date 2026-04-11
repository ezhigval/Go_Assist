package databases

import (
	"reflect"
	"testing"
)

func TestSingleScopeAccessNormalizesScope(t *testing.T) {
	t.Parallel()

	access, err := singleScopeAccess("BUSINESS")
	if err != nil {
		t.Fatalf("singleScopeAccess returned error: %v", err)
	}

	want := []string{"business"}
	if !reflect.DeepEqual(access.AllowedScopes, want) {
		t.Fatalf("AllowedScopes = %v, want %v", access.AllowedScopes, want)
	}
	if access.Bypass {
		t.Fatal("expected non-bypass access")
	}
}

func TestNewScopeAccessRejectsInvalidScopes(t *testing.T) {
	t.Parallel()

	if _, err := newScopeAccess([]string{"unknown"}); err == nil {
		t.Fatal("expected invalid scope error")
	}
}

func TestJournalScopeAccessSupportsUnrestrictedFilter(t *testing.T) {
	t.Parallel()

	access, err := journalScopeAccess(FullJournalScopeFilter())
	if err != nil {
		t.Fatalf("journalScopeAccess returned error: %v", err)
	}
	if !access.Bypass {
		t.Fatal("expected bypass access for unrestricted filter")
	}
	if len(access.AllowedScopes) != 0 {
		t.Fatalf("expected no explicit scopes, got %v", access.AllowedScopes)
	}
}

func TestNormalizeScopesDeduplicatesAndSorts(t *testing.T) {
	t.Parallel()

	got := normalizeScopes([]string{"travel", "business", "TRAVEL", "unknown", " business "})
	want := []string{"business", "travel"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeScopes() = %v, want %v", got, want)
	}
}
