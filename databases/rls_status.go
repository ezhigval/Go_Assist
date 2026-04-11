package databases

import (
	"context"
	"fmt"
)

// TableRLSStatus описывает RLS-конфигурацию одной таблицы.
type TableRLSStatus struct {
	TableName       string `json:"table_name"`
	TableRLSEnabled bool   `json:"table_rls_enabled"`
	TableRLSForced  bool   `json:"table_rls_forced"`
	SelectPolicy    bool   `json:"select_policy"`
	InsertPolicy    bool   `json:"insert_policy"`
}

// Effective сообщает, что table-level RLS включён и текущая DB role не обходит policy.
func (s TableRLSStatus) Effective(roleSuperuser, roleBypassRLS bool) bool {
	return s.TableName != "" &&
		!roleSuperuser &&
		!roleBypassRLS &&
		s.TableRLSEnabled &&
		s.TableRLSForced &&
		s.SelectPolicy &&
		s.InsertPolicy
}

func (s TableRLSStatus) warnings(roleSuperuser, roleBypassRLS bool) []string {
	var warnings []string
	if !s.TableRLSEnabled {
		warnings = append(warnings, fmt.Sprintf("%s row security is disabled", s.TableName))
	}
	if !s.TableRLSForced {
		warnings = append(warnings, fmt.Sprintf("%s does not use FORCE ROW LEVEL SECURITY", s.TableName))
	}
	if !s.SelectPolicy {
		warnings = append(warnings, fmt.Sprintf("%s select policy is missing", s.TableName))
	}
	if !s.InsertPolicy {
		warnings = append(warnings, fmt.Sprintf("%s insert policy is missing", s.TableName))
	}
	if roleSuperuser {
		warnings = append(warnings, fmt.Sprintf("%s RLS is bypassed because current role is PostgreSQL superuser", s.TableName))
	}
	if roleBypassRLS {
		warnings = append(warnings, fmt.Sprintf("%s RLS is bypassed because current role has BYPASSRLS", s.TableName))
	}
	return warnings
}

// StorageRLSStatus описывает RLS-готовность текущей DB role для защищённых storage/auth таблиц.
type StorageRLSStatus struct {
	CurrentUser   string         `json:"current_user"`
	RoleSuperuser bool           `json:"role_superuser"`
	RoleBypassRLS bool           `json:"role_bypass_rls"`
	Journal       TableRLSStatus `json:"journal"`
	Stats         TableRLSStatus `json:"stats"`
	Sessions      TableRLSStatus `json:"sessions"`
	AuthSessions  TableRLSStatus `json:"auth_sessions"`
}

// Effective сообщает, что все текущие storage/auth таблицы реально защищены для текущей DB role.
func (s StorageRLSStatus) Effective() bool {
	return s.CurrentUser != "" &&
		s.Journal.Effective(s.RoleSuperuser, s.RoleBypassRLS) &&
		s.Stats.Effective(s.RoleSuperuser, s.RoleBypassRLS) &&
		s.Sessions.Effective(s.RoleSuperuser, s.RoleBypassRLS) &&
		s.AuthSessions.Effective(s.RoleSuperuser, s.RoleBypassRLS)
}

// Warnings возвращает все причины, по которым storage RLS может быть неэффективен.
func (s StorageRLSStatus) Warnings() []string {
	var warnings []string
	if s.CurrentUser == "" {
		warnings = append(warnings, "current_user is empty; could not verify storage RLS context")
	}
	warnings = append(warnings, s.Journal.warnings(s.RoleSuperuser, s.RoleBypassRLS)...)
	warnings = append(warnings, s.Stats.warnings(s.RoleSuperuser, s.RoleBypassRLS)...)
	warnings = append(warnings, s.Sessions.warnings(s.RoleSuperuser, s.RoleBypassRLS)...)
	warnings = append(warnings, s.AuthSessions.warnings(s.RoleSuperuser, s.RoleBypassRLS)...)
	return warnings
}

// InspectStorageRLS читает метаданные PostgreSQL и возвращает готовность DB-enforced policy
// для текущих scope-bound и auth-bound таблиц (`event_journal`, `stats`, `sessions`, `auth_sessions`).
func (db *DB) InspectStorageRLS(ctx context.Context) (StorageRLSStatus, error) {
	status := StorageRLSStatus{}
	err := db.pool.QueryRow(ctx, `
		SELECT current_user, r.rolsuper, r.rolbypassrls
		FROM pg_roles r
		WHERE r.rolname = current_user
	`).Scan(&status.CurrentUser, &status.RoleSuperuser, &status.RoleBypassRLS)
	if err != nil {
		return StorageRLSStatus{}, fmt.Errorf("inspect storage RLS role status: %w", err)
	}

	journal, err := db.inspectTableRLS(ctx, "event_journal", "event_journal_scope_select", "event_journal_scope_insert")
	if err != nil {
		return StorageRLSStatus{}, err
	}
	stats, err := db.inspectTableRLS(ctx, "stats", "stats_scope_select", "stats_scope_insert")
	if err != nil {
		return StorageRLSStatus{}, err
	}
	sessions, err := db.inspectTableRLS(ctx, "sessions", "sessions_chat_select", "sessions_chat_write")
	if err != nil {
		return StorageRLSStatus{}, err
	}
	authSessions, err := db.inspectTableRLS(ctx, "auth_sessions", "auth_sessions_token_select", "auth_sessions_token_write")
	if err != nil {
		return StorageRLSStatus{}, err
	}
	status.Journal = journal
	status.Stats = stats
	status.Sessions = sessions
	status.AuthSessions = authSessions
	return status, nil
}

// InspectJournalRLS сохраняется как compatibility helper поверх общего inspector.
func (db *DB) InspectJournalRLS(ctx context.Context) (TableRLSStatus, error) {
	return db.inspectTableRLS(ctx, "event_journal", "event_journal_scope_select", "event_journal_scope_insert")
}

func (db *DB) inspectTableRLS(ctx context.Context, tableName, selectPolicy, insertPolicy string) (TableRLSStatus, error) {
	status := TableRLSStatus{TableName: tableName}
	err := db.pool.QueryRow(ctx, `
		SELECT
			c.relrowsecurity,
			c.relforcerowsecurity,
			EXISTS (
				SELECT 1
				FROM pg_policies p
				WHERE p.schemaname = n.nspname
				  AND p.tablename = c.relname
				  AND p.policyname = $3
			),
			EXISTS (
				SELECT 1
				FROM pg_policies p
				WHERE p.schemaname = n.nspname
				  AND p.tablename = c.relname
				  AND p.policyname = $4
			)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relname = $2
	`, "public", tableName, selectPolicy, insertPolicy).Scan(
		&status.TableRLSEnabled,
		&status.TableRLSForced,
		&status.SelectPolicy,
		&status.InsertPolicy,
	)
	if err != nil {
		return TableRLSStatus{}, fmt.Errorf("inspect table RLS %s: %w", tableName, err)
	}
	return status, nil
}
