package databases

import (
	"fmt"
	"regexp"
	"strings"
)

var pgIdentifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// BuildAppRoleBootstrapSQL печатает минимальный bootstrap для non-superuser app role,
// под которой journal RLS действительно будет enforceиться.
func BuildAppRoleBootstrapSQL(role, databaseName, schema string) (string, error) {
	role = normalizePGIdentifier(role)
	databaseName = normalizePGIdentifier(databaseName)
	schema = normalizePGIdentifier(schema)

	if !pgIdentifierPattern.MatchString(role) {
		return "", fmt.Errorf("build app role SQL: invalid role %q", role)
	}
	if !pgIdentifierPattern.MatchString(databaseName) {
		return "", fmt.Errorf("build app role SQL: invalid database %q", databaseName)
	}
	if !pgIdentifierPattern.MatchString(schema) {
		return "", fmt.Errorf("build app role SQL: invalid schema %q", schema)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "-- Bootstrap non-superuser app role for Modulr journal RLS\n")
	fmt.Fprintf(&b, "-- Run as database owner or administrative role, then configure DB_USER=%s\n\n", role)
	fmt.Fprintf(&b, "DO $$\nBEGIN\n")
	fmt.Fprintf(&b, "    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN\n", role)
	fmt.Fprintf(&b, "        CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;\n", role)
	fmt.Fprintf(&b, "    END IF;\n")
	fmt.Fprintf(&b, "END $$;\n\n")
	fmt.Fprintf(&b, "GRANT CONNECT ON DATABASE %s TO %s;\n", databaseName, role)
	fmt.Fprintf(&b, "GRANT USAGE ON SCHEMA %s TO %s;\n", schema, role)
	fmt.Fprintf(&b, "GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users, chats, sessions, stats TO %s;\n", role)
	fmt.Fprintf(&b, "GRANT SELECT, INSERT ON TABLE event_journal TO %s;\n", role)
	fmt.Fprintf(&b, "GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA %s TO %s;\n", schema, role)
	fmt.Fprintf(&b, "ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s;\n", schema, role)
	fmt.Fprintf(&b, "ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT USAGE, SELECT ON SEQUENCES TO %s;\n\n", schema, role)
	fmt.Fprintf(&b, "-- Optional hardening:\n")
	fmt.Fprintf(&b, "-- ALTER ROLE %s PASSWORD '<set-secure-password>';\n", role)
	fmt.Fprintf(&b, "-- Verify after switching DB_USER: go run ./cmd/databases rls-status\n")
	return b.String(), nil
}

func normalizePGIdentifier(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}
