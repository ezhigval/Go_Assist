package main

import (
	"reflect"
	"testing"
)

func TestBuildJournalFilterRequiresScopeWithoutAllScopes(t *testing.T) {
	t.Parallel()

	if _, err := buildJournalFilter("", "", false); err == nil {
		t.Fatal("expected missing scope error")
	}
}

func TestBuildJournalFilterBuildsScopedFilter(t *testing.T) {
	t.Parallel()

	filter, err := buildJournalFilter("personal", "business,travel", false)
	if err != nil {
		t.Fatalf("buildJournalFilter returned error: %v", err)
	}

	want := []string{"personal", "business", "travel"}
	if got := filter.Scopes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Scopes() = %v, want %v", got, want)
	}
}

func TestBuildJournalFilterRejectsMixedAllScopesFlags(t *testing.T) {
	t.Parallel()

	if _, err := buildJournalFilter("personal", "", true); err == nil {
		t.Fatal("expected mixed all-scopes validation error")
	}
}

func TestParseCSVScopesNormalizesAndDeduplicates(t *testing.T) {
	t.Parallel()

	got := parseCSVScopes(" business,travel,BUSINESS, ,travel ")
	want := []string{"business", "travel"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCSVScopes() = %v, want %v", got, want)
	}
}
