package databases

import (
	"reflect"
	"testing"
)

func TestStorageRLSStatusEffective(t *testing.T) {
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
	}
	if !status.Effective() {
		t.Fatalf("expected status to be effective: %+v", status)
	}
}

func TestStorageRLSStatusWarnings(t *testing.T) {
	t.Parallel()

	status := StorageRLSStatus{
		CurrentUser:   "postgres",
		RoleSuperuser: true,
		RoleBypassRLS: true,
		Journal: TableRLSStatus{
			TableName:       "event_journal",
			TableRLSEnabled: false,
			TableRLSForced:  false,
			SelectPolicy:    false,
			InsertPolicy:    false,
		},
		Stats: TableRLSStatus{
			TableName:       "stats",
			TableRLSEnabled: false,
			TableRLSForced:  false,
			SelectPolicy:    false,
			InsertPolicy:    false,
		},
	}

	want := []string{
		"event_journal row security is disabled",
		"event_journal does not use FORCE ROW LEVEL SECURITY",
		"event_journal select policy is missing",
		"event_journal insert policy is missing",
		"event_journal RLS is bypassed because current role is PostgreSQL superuser",
		"event_journal RLS is bypassed because current role has BYPASSRLS",
		"stats row security is disabled",
		"stats does not use FORCE ROW LEVEL SECURITY",
		"stats select policy is missing",
		"stats insert policy is missing",
		"stats RLS is bypassed because current role is PostgreSQL superuser",
		"stats RLS is bypassed because current role has BYPASSRLS",
	}
	if got := status.Warnings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Warnings() = %v, want %v", got, want)
	}
}
