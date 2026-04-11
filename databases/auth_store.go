package databases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"modulr/auth"
	"modulr/events"

	"github.com/jackc/pgx/v5"
)

// AuthSessionStore хранит auth.Session в PostgreSQL через auth_sessions.
type AuthSessionStore struct {
	db *DB
}

var _ auth.SessionStore = (*AuthSessionStore)(nil)

// NewAuthSessionStore создаёт DB-backed storage для модуля auth.
func NewAuthSessionStore(db *DB) *AuthSessionStore {
	return &AuthSessionStore{db: db}
}

type authSessionRecord struct {
	TokenHash     string
	UserID        string
	Scope         string
	AllowedScopes []string
	Roles         []auth.Role
	Meta          map[string]interface{}
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// Put сохраняет auth-сессию по hash(token).
func (s *AuthSessionStore) Put(ctx context.Context, token string, sess *auth.Session) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("auth: database session store is nil")
	}

	record, access, err := buildAuthSessionRecord(token, sess)
	if err != nil {
		return err
	}

	allowedScopesBytes, err := json.Marshal(record.AllowedScopes)
	if err != nil {
		return fmt.Errorf("marshal auth allowed scopes: %w", err)
	}
	rolesBytes, err := json.Marshal(rolesToStrings(record.Roles))
	if err != nil {
		return fmt.Errorf("marshal auth roles: %w", err)
	}
	metaBytes, err := marshalJSONMap(record.Meta)
	if err != nil {
		return fmt.Errorf("marshal auth meta: %w", err)
	}

	return s.db.withStorageAccess(ctx, access, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO auth_sessions (token_hash, user_id, scope, allowed_scopes, roles, meta, created_at, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (token_hash) DO UPDATE
			SET user_id = EXCLUDED.user_id,
			    scope = EXCLUDED.scope,
			    allowed_scopes = EXCLUDED.allowed_scopes,
			    roles = EXCLUDED.roles,
			    meta = EXCLUDED.meta,
			    created_at = EXCLUDED.created_at,
			    expires_at = EXCLUDED.expires_at
		`, record.TokenHash, record.UserID, record.Scope, allowedScopesBytes, rolesBytes, metaBytes, record.CreatedAt, record.ExpiresAt)
		return err
	})
}

// Get загружает auth-сессию по исходному токену.
func (s *AuthSessionStore) Get(ctx context.Context, token string) (*auth.Session, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("auth: database session store is nil")
	}

	tokenHash := hashAuthToken(token)
	access, err := authTokenAccess(tokenHash, nil)
	if err != nil {
		return nil, err
	}

	record := authSessionRecord{TokenHash: tokenHash}
	var allowedScopesBytes []byte
	var rolesBytes []byte
	var metaBytes []byte

	err = s.db.withStorageAccess(ctx, access, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT user_id, scope, allowed_scopes, roles, meta, created_at, expires_at
			FROM auth_sessions
			WHERE token_hash = $1
		`, tokenHash).Scan(
			&record.UserID,
			&record.Scope,
			&allowedScopesBytes,
			&rolesBytes,
			&metaBytes,
			&record.CreatedAt,
			&record.ExpiresAt,
		)
	})
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("auth: session not found")
	}
	if err != nil {
		return nil, err
	}

	record.AllowedScopes, err = unmarshalStringSlice(allowedScopesBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal auth allowed scopes: %w", err)
	}
	record.Roles, err = unmarshalAuthRoles(rolesBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal auth roles: %w", err)
	}
	record.Meta, err = unmarshalJSONMap(metaBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal auth meta: %w", err)
	}
	return hydrateAuthSession(token, record), nil
}

// Delete инвалидирует auth-сессию по токену.
func (s *AuthSessionStore) Delete(ctx context.Context, token string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("auth: database session store is nil")
	}

	tokenHash := hashAuthToken(token)
	access, err := authTokenAccess(tokenHash, nil)
	if err != nil {
		return err
	}

	return s.db.withStorageAccess(ctx, access, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM auth_sessions WHERE token_hash = $1", tokenHash)
		return err
	})
}

func buildAuthSessionRecord(token string, sess *auth.Session) (authSessionRecord, storageAccess, error) {
	if strings.TrimSpace(token) == "" || sess == nil {
		return authSessionRecord{}, storageAccess{}, fmt.Errorf("auth: invalid session put")
	}
	if strings.TrimSpace(sess.UserID) == "" {
		return authSessionRecord{}, storageAccess{}, fmt.Errorf("auth: empty user id")
	}

	scope := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(sess.Scope)))
	if scope == "" {
		scope = events.DefaultSegment()
	}

	allowedScopes := normalizeScopes(append(append([]string(nil), sess.AllowedScopes...), string(scope)))
	if len(allowedScopes) == 0 {
		allowedScopes = []string{string(scope)}
	}

	tokenHash := hashAuthToken(token)
	access, err := authTokenAccess(tokenHash, allowedScopes)
	if err != nil {
		return authSessionRecord{}, storageAccess{}, err
	}

	return authSessionRecord{
		TokenHash:     tokenHash,
		UserID:        sess.UserID,
		Scope:         string(scope),
		AllowedScopes: allowedScopes,
		Roles:         append([]auth.Role(nil), sess.Roles...),
		Meta:          cloneJSONMap(sess.Meta),
		CreatedAt:     sess.CreatedAt,
		ExpiresAt:     sess.ExpiresAt,
	}, access, nil
}

func hydrateAuthSession(token string, record authSessionRecord) *auth.Session {
	return &auth.Session{
		Token:         token,
		UserID:        record.UserID,
		Scope:         record.Scope,
		AllowedScopes: append([]string(nil), record.AllowedScopes...),
		Roles:         append([]auth.Role(nil), record.Roles...),
		CreatedAt:     record.CreatedAt,
		ExpiresAt:     record.ExpiresAt,
		Meta:          cloneJSONMap(record.Meta),
	}
}

func hashAuthToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func unmarshalStringSlice(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func unmarshalAuthRoles(raw []byte) ([]auth.Role, error) {
	items, err := unmarshalStringSlice(raw)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]auth.Role, 0, len(items))
	for _, item := range items {
		out = append(out, auth.Role(item))
	}
	return out, nil
}

func rolesToStrings(items []auth.Role) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, string(item))
	}
	return out
}

func cloneJSONMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
