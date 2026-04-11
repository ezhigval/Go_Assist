package state

import "testing"

func TestSetActiveScopePersistsValidScope(t *testing.T) {
	t.Parallel()

	session := SetActiveScope(Session{}, "business")
	if got := ActiveScope(session); got != "business" {
		t.Fatalf("ActiveScope() = %q, want business", got)
	}
}

func TestSetActiveScopeDropsInvalidScope(t *testing.T) {
	t.Parallel()

	session := SetActiveScope(Session{
		Payload: map[string]interface{}{"_active_scope": "travel", "draft": "note"},
	}, "unknown")

	if got := ActiveScope(session); got != "" {
		t.Fatalf("ActiveScope() = %q, want empty", got)
	}
	if session.Payload["draft"] != "note" {
		t.Fatalf("expected unrelated payload to survive, got %+v", session.Payload)
	}
}

func TestPreserveActiveScopeCopiesCurrentScopeToNextState(t *testing.T) {
	t.Parallel()

	current := SetActiveScope(Session{}, "health")
	next := PreserveActiveScope(current, Session{Key: "awaiting_input"})

	if got := ActiveScope(next); got != "health" {
		t.Fatalf("ActiveScope() = %q, want health", got)
	}
	if next.Key != "awaiting_input" {
		t.Fatalf("expected next key to stay intact, got %q", next.Key)
	}
}
