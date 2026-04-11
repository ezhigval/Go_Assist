package events

import "strings"

const allowScopeTagPrefix = "allow_scope:"

// ScopeAllowed проверяет, можно ли обрабатывать candidateScope в рамках baseScope.
// По умолчанию разрешён только same-scope путь. Cross-scope допускается
// только при явном перечислении target scope в tags (`allow_scope:<segment>`)
// или metadata (`allowed_scopes`).
func ScopeAllowed(baseScope, candidateScope Segment, tags []string, metadata map[string]any) bool {
	if candidateScope == "" {
		candidateScope = baseScope
	}
	if !IsValidSegment(candidateScope) {
		return false
	}
	if baseScope == "" {
		return true
	}
	if !IsValidSegment(baseScope) {
		return false
	}
	if candidateScope == baseScope {
		return true
	}

	allowed := allowedScopeSet(tags, metadata)
	_, ok := allowed[candidateScope]
	return ok
}

func allowedScopeSet(tags []string, metadata map[string]any) map[Segment]struct{} {
	out := make(map[Segment]struct{})
	for _, scope := range segmentsFromAny(metadata["allowed_scopes"]) {
		out[scope] = struct{}{}
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if !strings.HasPrefix(tag, allowScopeTagPrefix) {
			continue
		}
		scope := ParseSegmentFromAny(strings.TrimPrefix(tag, allowScopeTagPrefix))
		if scope == "" {
			continue
		}
		out[scope] = struct{}{}
	}
	return out
}

func segmentsFromAny(v any) []Segment {
	switch x := v.(type) {
	case []Segment:
		out := make([]Segment, 0, len(x))
		for _, item := range x {
			if IsValidSegment(item) {
				out = append(out, item)
			}
		}
		return out
	case []string:
		out := make([]Segment, 0, len(x))
		for _, item := range x {
			if scope := ParseSegmentFromAny(item); scope != "" {
				out = append(out, scope)
			}
		}
		return out
	case []any:
		out := make([]Segment, 0, len(x))
		for _, item := range x {
			if scope := ParseSegmentFromAny(item); scope != "" {
				out = append(out, scope)
			}
		}
		return out
	case string:
		if scope := ParseSegmentFromAny(x); scope != "" {
			return []Segment{scope}
		}
	}
	return nil
}
