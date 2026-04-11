package databases

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"modulr/events"

	"github.com/jackc/pgx/v5"
)

const (
	rlsAllowedScopesSetting = "modulr.allowed_scopes"
	rlsScopeBypassSetting   = "modulr.scope_bypass"
	rlsChatIDSetting        = "modulr.chat_id"
	rlsAuthTokenHashSetting = "modulr.auth_token_hash"
)

type storageAccess struct {
	AllowedScopes []string
	Bypass        bool
	ChatID        *int64
	AuthTokenHash string
}

func journalScopeAccess(filter JournalScopeFilter) (storageAccess, error) {
	if err := filter.validate(); err != nil {
		return storageAccess{}, err
	}
	if filter.Unrestricted() {
		return storageAccess{Bypass: true}, nil
	}
	return newScopeAccess(filter.Scopes())
}

func singleScopeAccess(scope string) (storageAccess, error) {
	return newScopeAccess([]string{scope})
}

func newScopeAccess(scopes []string) (storageAccess, error) {
	normalized := normalizeScopes(scopes)
	if len(normalized) == 0 {
		return storageAccess{}, fmt.Errorf("scope access: at least one valid scope is required")
	}
	return storageAccess{AllowedScopes: normalized}, nil
}

func sessionReadAccess(chatID int64) storageAccess {
	return storageAccess{ChatID: &chatID}
}

func sessionWriteAccess(chatID int64, scope string) (storageAccess, error) {
	access, err := singleScopeAccess(scope)
	if err != nil {
		return storageAccess{}, err
	}
	access.ChatID = &chatID
	return access, nil
}

func authTokenAccess(tokenHash string, allowedScopes []string) (storageAccess, error) {
	tokenHash = strings.TrimSpace(strings.ToLower(tokenHash))
	if tokenHash == "" {
		return storageAccess{}, fmt.Errorf("auth token access: token hash is required")
	}

	access := storageAccess{AuthTokenHash: tokenHash}
	if len(allowedScopes) == 0 {
		return access, nil
	}

	normalized := normalizeScopes(allowedScopes)
	if len(normalized) == 0 {
		return storageAccess{}, fmt.Errorf("auth token access: at least one valid scope is required when allowed scopes are provided")
	}
	access.AllowedScopes = normalized
	return access, nil
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

func (a storageAccess) scopeList() string {
	if len(a.AllowedScopes) == 0 {
		return ""
	}
	return strings.Join(a.AllowedScopes, ",")
}

func (a storageAccess) chatIDValue() string {
	if a.ChatID == nil {
		return ""
	}
	return strconv.FormatInt(*a.ChatID, 10)
}

func (db *DB) withStorageAccess(ctx context.Context, access storageAccess, fn func(tx pgx.Tx) error) (err error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin storage access tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = applyStorageAccess(ctx, tx, access); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit storage access tx: %w", err)
	}
	return nil
}

func (db *DB) withScopeAccess(ctx context.Context, access storageAccess, fn func(tx pgx.Tx) error) error {
	return db.withStorageAccess(ctx, access, fn)
}

func applyStorageAccess(ctx context.Context, tx pgx.Tx, access storageAccess) error {
	if !access.Bypass && len(access.AllowedScopes) == 0 && access.ChatID == nil && access.AuthTokenHash == "" {
		return fmt.Errorf("apply storage access: at least one guard is required unless bypass is enabled")
	}

	if _, err := tx.Exec(ctx, "SET LOCAL row_security = on"); err != nil {
		return fmt.Errorf("apply storage access: enable row_security: %w", err)
	}
	bypassValue := "off"
	if access.Bypass {
		bypassValue = "on"
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsScopeBypassSetting, bypassValue); err != nil {
		return fmt.Errorf("apply storage access: set %s: %w", rlsScopeBypassSetting, err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsAllowedScopesSetting, access.scopeList()); err != nil {
		return fmt.Errorf("apply storage access: set %s: %w", rlsAllowedScopesSetting, err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsChatIDSetting, access.chatIDValue()); err != nil {
		return fmt.Errorf("apply storage access: set %s: %w", rlsChatIDSetting, err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", rlsAuthTokenHashSetting, access.AuthTokenHash); err != nil {
		return fmt.Errorf("apply storage access: set %s: %w", rlsAuthTokenHashSetting, err)
	}
	return nil
}

func applyScopeAccess(ctx context.Context, tx pgx.Tx, access storageAccess) error {
	return applyStorageAccess(ctx, tx, access)
}
