package databases

import (
	"strings"
	"testing"
)

func TestEnforceStorageRLSAllowsEffectiveStatus(t *testing.T) {
	t.Parallel()

	status := StorageRLSStatus{
		CurrentUser: "modulr_app",
		Journal: TableRLSStatus{
			TableName:       "event_journal",
			TableRLSEnabled: true,
			TableRLSForced:  true,
			SelectPolicy:    true,
			InsertPolicy:    true,
		},
		Stats: TableRLSStatus{
			TableName:       "stats",
			TableRLSEnabled: true,
			TableRLSForced:  true,
			SelectPolicy:    true,
			InsertPolicy:    true,
		},
		Sessions: TableRLSStatus{
			TableName:       "sessions",
			TableRLSEnabled: true,
			TableRLSForced:  true,
			SelectPolicy:    true,
			InsertPolicy:    true,
		},
		AuthSessions: TableRLSStatus{
			TableName:       "auth_sessions",
			TableRLSEnabled: true,
			TableRLSForced:  true,
			SelectPolicy:    true,
			InsertPolicy:    true,
		},
	}

	if err := EnforceStorageRLS(status, true); err != nil {
		t.Fatalf("EnforceStorageRLS returned error: %v", err)
	}
}

func TestEnforceStorageRLSRejectsIneffectiveStatus(t *testing.T) {
	t.Parallel()

	status := StorageRLSStatus{
		CurrentUser: "postgres",
		Journal: TableRLSStatus{
			TableName: "event_journal",
		},
		Stats: TableRLSStatus{
			TableName: "stats",
		},
		Sessions: TableRLSStatus{
			TableName: "sessions",
		},
		AuthSessions: TableRLSStatus{
			TableName: "auth_sessions",
		},
	}

	err := EnforceStorageRLS(status, true)
	if err == nil {
		t.Fatal("expected enforcement error")
	}
	if !strings.Contains(err.Error(), "storage RLS is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
