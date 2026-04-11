package databases

import (
	"strings"
	"testing"
	"time"

	"modulr/auth"
)

func TestBuildAuthSessionRecordNormalizesScopeAndHash(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	record, access, err := buildAuthSessionRecord(" raw-token ", &auth.Session{
		UserID:        "user-1",
		Scope:         "BUSINESS",
		AllowedScopes: []string{"travel", "business", "unknown"},
		Roles:         []auth.Role{auth.RoleAdmin, auth.RoleUser},
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		Meta:          map[string]interface{}{"source": "telegram"},
	})
	if err != nil {
		t.Fatalf("buildAuthSessionRecord returned error: %v", err)
	}
	if record.Scope != "business" {
		t.Fatalf("scope = %q, want business", record.Scope)
	}
	wantScopes := []string{"business", "travel"}
	for i, scope := range wantScopes {
		if record.AllowedScopes[i] != scope {
			t.Fatalf("allowed scopes = %v, want %v", record.AllowedScopes, wantScopes)
		}
	}
	if strings.Contains(record.TokenHash, "raw-token") {
		t.Fatalf("token hash leaked plaintext token: %q", record.TokenHash)
	}
	if access.AuthTokenHash != record.TokenHash {
		t.Fatalf("access token hash = %q, want %q", access.AuthTokenHash, record.TokenHash)
	}
}

func TestHydrateAuthSessionRestoresSession(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	sess := hydrateAuthSession("plain-token", authSessionRecord{
		UserID:        "user-1",
		Scope:         "family",
		AllowedScopes: []string{"family", "travel"},
		Roles:         []auth.Role{auth.RoleUser},
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		Meta:          map[string]interface{}{"source": "telegram"},
	})

	if sess.Token != "plain-token" {
		t.Fatalf("token = %q, want plain-token", sess.Token)
	}
	if sess.Scope != "family" {
		t.Fatalf("scope = %q, want family", sess.Scope)
	}
	if len(sess.AllowedScopes) != 2 || sess.AllowedScopes[1] != "travel" {
		t.Fatalf("allowed scopes = %v, want [family travel]", sess.AllowedScopes)
	}
}

func TestHashAuthTokenDeterministic(t *testing.T) {
	t.Parallel()

	first := hashAuthToken("token-123")
	second := hashAuthToken("token-123")
	third := hashAuthToken("token-456")

	if first != second {
		t.Fatalf("expected deterministic hash, got %q and %q", first, second)
	}
	if first == third {
		t.Fatalf("expected different tokens to hash differently, got %q", first)
	}
}
