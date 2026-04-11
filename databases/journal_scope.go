package databases

import (
	"fmt"
	"sort"

	"modulr/events"
)

// JournalScopeFilter ограничивает read-path event_journal разрешёнными scope.
// AllowedScopes всегда содержит базовый scope; unrestricted=true оставляет доступ
// к полному журналу для admin/deploy tooling.
type JournalScopeFilter struct {
	BaseScope     string
	AllowedScopes []string
	unrestricted  bool
}

// NewJournalScopeFilter строит read-policy для event_journal по тем же правилам,
// что и runtime-level scope isolation (`allowed_scopes`, `allow_scope:<segment>`).
func NewJournalScopeFilter(baseScope string, tags []string, metadata map[string]any) (JournalScopeFilter, error) {
	base := events.ParseSegmentFromAny(baseScope)
	if base == "" {
		return JournalScopeFilter{}, fmt.Errorf("journal scope filter: invalid base scope %q", baseScope)
	}

	extras := make([]string, 0)
	for _, candidate := range events.AllSegments() {
		if candidate == base {
			continue
		}
		if events.ScopeAllowed(base, candidate, tags, metadata) {
			extras = append(extras, string(candidate))
		}
	}
	sort.Strings(extras)

	allowed := make([]string, 0, 1+len(extras))
	allowed = append(allowed, string(base))
	allowed = append(allowed, extras...)

	return JournalScopeFilter{
		BaseScope:     string(base),
		AllowedScopes: allowed,
	}, nil
}

// FullJournalScopeFilter даёт явный unrestricted-доступ для migration/admin tooling.
func FullJournalScopeFilter() JournalScopeFilter {
	return JournalScopeFilter{unrestricted: true}
}

// Unrestricted сообщает, что filter не режет журнал по scope.
func (f JournalScopeFilter) Unrestricted() bool {
	return f.unrestricted
}

// Scopes возвращает копию разрешённых scope; для unrestricted фильтра вернёт nil.
func (f JournalScopeFilter) Scopes() []string {
	if f.unrestricted || len(f.AllowedScopes) == 0 {
		return nil
	}
	out := make([]string, len(f.AllowedScopes))
	copy(out, f.AllowedScopes)
	return out
}

// Allows проверяет, разрешён ли scope политикой фильтра.
func (f JournalScopeFilter) Allows(scope string) bool {
	if f.unrestricted {
		return true
	}
	seg := events.ParseSegmentFromAny(scope)
	if seg == "" {
		return false
	}
	for _, item := range f.AllowedScopes {
		if item == string(seg) {
			return true
		}
	}
	return false
}

func (f JournalScopeFilter) validate() error {
	if f.unrestricted {
		return nil
	}
	base := events.ParseSegmentFromAny(f.BaseScope)
	if base == "" {
		return fmt.Errorf("journal scope filter: base scope is required; use FullJournalScopeFilter for unrestricted reads")
	}
	if len(f.AllowedScopes) == 0 {
		return fmt.Errorf("journal scope filter: allowed scopes are required")
	}
	if !f.Allows(string(base)) {
		return fmt.Errorf("journal scope filter: base scope %q must be part of allowed scopes", f.BaseScope)
	}
	for _, scope := range f.AllowedScopes {
		if events.ParseSegmentFromAny(scope) == "" {
			return fmt.Errorf("journal scope filter: invalid allowed scope %q", scope)
		}
	}
	return nil
}
