package events

import (
	"sort"
	"strings"
)

const (
	RoleAdminName = "admin"
	RoleUserName  = "user"
	RoleGuestName = "guest"
)

// NormalizeRoles приводит any к каноническому набору ролей.
func NormalizeRoles(v any) []string {
	var raw []string
	switch x := v.(type) {
	case []string:
		raw = append(raw, x...)
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok {
				raw = append(raw, s)
			}
		}
	case string:
		if x != "" {
			raw = append(raw, x)
		}
	}
	if len(raw) == 0 {
		return nil
	}

	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		role := strings.TrimSpace(strings.ToLower(item))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// RoleAllowsEvent возвращает true, если роль разрешает публикацию eventName.
func RoleAllowsEvent(role, eventName string) bool {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case RoleAdminName:
		return true
	case RoleUserName:
		return true
	case RoleGuestName:
		return eventName == string(V1SystemStartup)
	default:
		return false
	}
}

// RolesAllowEvent true, если хотя бы одна роль разрешает событие.
func RolesAllowEvent(roles []string, eventName string) bool {
	for _, role := range NormalizeRoles(roles) {
		if RoleAllowsEvent(role, eventName) {
			return true
		}
	}
	return false
}

// MetadataAuthRequired читает auth requirement из metadata.
func MetadataAuthRequired(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	switch v := metadata["auth_required"].(type) {
	case bool:
		return v
	case string:
		switch strings.TrimSpace(strings.ToLower(v)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}
