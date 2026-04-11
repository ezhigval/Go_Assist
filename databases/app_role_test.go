package databases

import (
	"strings"
	"testing"
)

func TestBuildAppRoleBootstrapSQL(t *testing.T) {
	t.Parallel()

	sql, err := BuildAppRoleBootstrapSQL("Modulr_App", "Telegram_Bot", "Public")
	if err != nil {
		t.Fatalf("BuildAppRoleBootstrapSQL returned error: %v", err)
	}

	wantSnippets := []string{
		"CREATE ROLE modulr_app LOGIN NOSUPERUSER",
		"GRANT CONNECT ON DATABASE telegram_bot TO modulr_app;",
		"GRANT SELECT, INSERT ON TABLE event_journal TO modulr_app;",
		"go run ./cmd/databases rls-status",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("generated SQL does not contain %q:\n%s", snippet, sql)
		}
	}
}

func TestBuildAppRoleBootstrapSQLRejectsInvalidIdentifier(t *testing.T) {
	t.Parallel()

	if _, err := BuildAppRoleBootstrapSQL("modulr-app", "telegram_bot", "public"); err == nil {
		t.Fatal("expected invalid role error")
	}
}
