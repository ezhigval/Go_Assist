package auth

import "time"

// Role роль пользователя для матрицы прав на события.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

// Session сессия после успешной аутентификации.
type Session struct {
	Token         string
	UserID        string
	Scope         string
	AllowedScopes []string
	Roles         []Role
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Meta          map[string]any
}
