package databases

import (
	"context"
	"fmt"
)

// JournalRLSStatus описывает, действительно ли текущая DB role подпадает под journal RLS.
type JournalRLSStatus struct {
	CurrentUser     string `json:"current_user"`
	RoleSuperuser   bool   `json:"role_superuser"`
	RoleBypassRLS   bool   `json:"role_bypass_rls"`
	TableRLSEnabled bool   `json:"table_rls_enabled"`
	TableRLSForced  bool   `json:"table_rls_forced"`
	SelectPolicy    bool   `json:"select_policy"`
	InsertPolicy    bool   `json:"insert_policy"`
}

// Effective сообщает, что journal RLS включён и текущая роль не обходит policy.
func (s JournalRLSStatus) Effective() bool {
	return s.CurrentUser != "" &&
		!s.RoleSuperuser &&
		!s.RoleBypassRLS &&
		s.TableRLSEnabled &&
		s.TableRLSForced &&
		s.SelectPolicy &&
		s.InsertPolicy
}

// Warnings возвращает человекочитаемые причины, почему RLS может не защищать данные.
func (s JournalRLSStatus) Warnings() []string {
	var warnings []string
	if s.CurrentUser == "" {
		warnings = append(warnings, "current_user is empty; could not verify journal RLS context")
	}
	if s.RoleSuperuser {
		warnings = append(warnings, "current role is PostgreSQL superuser and bypasses RLS")
	}
	if s.RoleBypassRLS {
		warnings = append(warnings, "current role has BYPASSRLS and bypasses journal policy")
	}
	if !s.TableRLSEnabled {
		warnings = append(warnings, "event_journal row security is disabled")
	}
	if !s.TableRLSForced {
		warnings = append(warnings, "event_journal does not use FORCE ROW LEVEL SECURITY")
	}
	if !s.SelectPolicy {
		warnings = append(warnings, "event_journal select policy is missing")
	}
	if !s.InsertPolicy {
		warnings = append(warnings, "event_journal insert policy is missing")
	}
	return warnings
}

// InspectJournalRLS читает метаданные PostgreSQL и возвращает готовность DB-enforced policy.
func (db *DB) InspectJournalRLS(ctx context.Context) (JournalRLSStatus, error) {
	status := JournalRLSStatus{}
	err := db.pool.QueryRow(ctx, `
		SELECT
			current_user,
			r.rolsuper,
			r.rolbypassrls,
			c.relrowsecurity,
			c.relforcerowsecurity,
			EXISTS (
				SELECT 1
				FROM pg_policies p
				WHERE p.schemaname = n.nspname
				  AND p.tablename = c.relname
				  AND p.policyname = 'event_journal_scope_select'
			),
			EXISTS (
				SELECT 1
				FROM pg_policies p
				WHERE p.schemaname = n.nspname
				  AND p.tablename = c.relname
				  AND p.policyname = 'event_journal_scope_insert'
			)
		FROM pg_roles r
		JOIN pg_class c ON c.oid = 'public.event_journal'::regclass
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE r.rolname = current_user
	`).Scan(
		&status.CurrentUser,
		&status.RoleSuperuser,
		&status.RoleBypassRLS,
		&status.TableRLSEnabled,
		&status.TableRLSForced,
		&status.SelectPolicy,
		&status.InsertPolicy,
	)
	if err != nil {
		return JournalRLSStatus{}, fmt.Errorf("inspect journal RLS: %w", err)
	}
	return status, nil
}
