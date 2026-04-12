package state

import "testing"

func TestSetAuthSessionReferencePersistsValue(t *testing.T) {
	t.Parallel()

	session := SetAuthSessionReference(Session{}, "Ref-123")
	if got := AuthSessionReference(session); got != "ref-123" {
		t.Fatalf("AuthSessionReference() = %q, want ref-123", got)
	}
}

func TestPreserveAuthSessionReferenceCopiesCurrentValue(t *testing.T) {
	t.Parallel()

	current := SetAuthSessionReference(Session{}, "ref-123")
	next := PreserveAuthSessionReference(current, Session{Key: "awaiting_input"})

	if got := AuthSessionReference(next); got != "ref-123" {
		t.Fatalf("AuthSessionReference() = %q, want ref-123", got)
	}
}

func TestPreserveAuthSessionReferenceHonorsExplicitClear(t *testing.T) {
	t.Parallel()

	current := SetAuthSessionReference(Session{}, "ref-123")
	next := PreserveAuthSessionReference(current, SetAuthSessionReference(Session{}, ""))

	if got := AuthSessionReference(next); got != "" {
		t.Fatalf("AuthSessionReference() = %q, want empty", got)
	}
}
