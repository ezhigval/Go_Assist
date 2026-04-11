package databases

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"modulr/events"

	"github.com/jackc/pgx/v5"
)

const (
	rlsAllowedScopesSetting = "modulr.allowed_scopes"
	rlsScopeBypassSetting   = "modulr.scope_bypass"
)

type scopeAccess struct {
	AllowedScopes []string
	Bypass        bool
}

func journalScopeAccess(filter JournalScopeFilter) (scopeAccess, error) {
	if err := filter.validate(); err != nil {
		return scopeAccess{}, err
	}
	if filter.Unrestricted() {
		return scopeAccess{Bypass: true}, nil
	}
	return newScopeAccess(filter.Scopes())
}

func singleScopeAccess(scope string) (scopeAccess, error) {
	return newScopeAccess([]string{scope})
}

func newScopeAccess(scopes []string) (scopeAccess, error) {
	normalized := normalizeScopes(scopes)
	if len(normalized) == 0 {
		return scopeAccess{}, fmt.Errorf("scope access: at least one valid scope is required")
	}
	return scopeAccess{AllowedScopes: normalized}, nil
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		seg := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(scope)))
		if seg == "" {
			continue
		}
		key := string(seg)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (a scopeAccess) scopeList() string {
	if len(a.AllowedScopes) == 0 {
		return ""
	}
	return strings.Join(a.AllowedScopes, ",")
}

func (db *DB) withScopeAccess(ctx context.Context, access scopeAccess, fn func(tx pgx.Tx) error) (err error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin scoped tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = applyScopeAccess(ctx, tx, access); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit scoped tx: %w", err)
	}
	return nil
}

func applyScopeAccess(ctx context.Context, tx pgx.Tx, access scopeAccess) error {
	if !access.Bypass && len(access.AllowedScopes) == 0 {
		return fmt.Errorf("apply scope access: allowed scopes required unless bypass is enabled")
	}

	if _, err := tx.Exec(ctx, "SET LOCAL row_security = on"); err != nil {
		return fmt.Errorf("apply scope access: enable row_security: %w", err)
	}
	bypassValue := "off"
	if access.Bypass {
		bypassValue = "on"
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsScopeBypassSetting, bypassValue); err != nil {
		return fmt.Errorf("apply scope access: set %s: %w", rlsScopeBypassSetting, err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsAllowedScopesSetting, access.scopeList()); err != nil {
		return fmt.Errorf("apply scope access: set %s: %w", rlsAllowedScopesSetting, err)
	}
	return nil
}
