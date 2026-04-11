package auth

import (
	"context"
	"sort"
	"strings"

	"modulr/events"
)

type sessionScopeContextKey struct{}
type allowedScopesContextKey struct{}

// WithSessionScope фиксирует базовый scope для новой auth-сессии.
func WithSessionScope(ctx context.Context, scope string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionScopeContextKey{}, scope)
}

// WithAllowedScopes задаёт явный список дополнительных scope для auth-сессии.
func WithAllowedScopes(ctx context.Context, scopes []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	copied := make([]string, len(scopes))
	copy(copied, scopes)
	return context.WithValue(ctx, allowedScopesContextKey{}, copied)
}

// ScopeAllowed сообщает, разрешён ли target scope для auth-сессии.
func ScopeAllowed(sess *Session, targetScope string) bool {
	if sess == nil {
		return false
	}
	target := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(targetScope)))
	if target == "" {
		return false
	}
	for _, scope := range sessionAllowedScopes(sess.Scope, sess.AllowedScopes) {
		if scope == string(target) {
			return true
		}
	}
	return false
}

func sessionAccessFromContext(ctx context.Context) (string, []string) {
	if ctx == nil {
		return string(events.DefaultSegment()), []string{string(events.DefaultSegment())}
	}

	scope, _ := ctx.Value(sessionScopeContextKey{}).(string)
	var allowed []string
	switch v := ctx.Value(allowedScopesContextKey{}).(type) {
	case []string:
		allowed = v
	}

	scope = normalizeSessionScope(scope)
	return scope, sessionAllowedScopes(scope, allowed)
}

func normalizeSessionScope(scope string) string {
	seg := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(scope)))
	if seg == "" {
		seg = events.DefaultSegment()
	}
	return string(seg)
}

func sessionAllowedScopes(baseScope string, allowed []string) []string {
	normalized := make([]string, 0, len(allowed)+1)
	seen := make(map[string]struct{}, len(allowed)+1)

	base := normalizeSessionScope(baseScope)
	seen[base] = struct{}{}
	normalized = append(normalized, base)

	for _, scope := range allowed {
		seg := events.ParseSegmentFromAny(strings.TrimSpace(strings.ToLower(scope)))
		if seg == "" {
			continue
		}
		key := string(seg)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}

	sort.Strings(normalized[1:])
	return normalized
}
