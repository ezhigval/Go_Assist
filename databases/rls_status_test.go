package databases

import (
	"reflect"
	"testing"
)

func TestJournalRLSStatusEffective(t *testing.T) {
	t.Parallel()

	status := JournalRLSStatus{
		CurrentUser:     "modulr_app",
		TableRLSEnabled: true,
		TableRLSForced:  true,
		SelectPolicy:    true,
		InsertPolicy:    true,
	}
	if !status.Effective() {
		t.Fatalf("expected status to be effective: %+v", status)
	}
}

func TestJournalRLSStatusWarnings(t *testing.T) {
	t.Parallel()

	status := JournalRLSStatus{
		CurrentUser:     "postgres",
		RoleSuperuser:   true,
		RoleBypassRLS:   true,
		TableRLSEnabled: false,
		TableRLSForced:  false,
		SelectPolicy:    false,
		InsertPolicy:    false,
	}

	want := []string{
		"current role is PostgreSQL superuser and bypasses RLS",
		"current role has BYPASSRLS and bypasses journal policy",
		"event_journal row security is disabled",
		"event_journal does not use FORCE ROW LEVEL SECURITY",
		"event_journal select policy is missing",
		"event_journal insert policy is missing",
	}
	if got := status.Warnings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Warnings() = %v, want %v", got, want)
	}
}
