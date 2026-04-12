package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	modulrauth "modulr/auth"
	"telegram/handler"
	"telegram/state"
)

// AuthConfig настраивает auth-gating для transport ingress.
type AuthConfig struct {
	Required      bool
	AdminUserIDs  []int64
	AllowedScopes []string
}

// IngressOption конфигурирует дополнительные зависимости transport ingress.
type IngressOption func(*ingressConfig)

type ingressConfig struct {
	auth    modulrauth.AuthAPI
	authCfg AuthConfig
}

type ingressAuthResult struct {
	Session   *modulrauth.Session
	NextState state.Session
	Response  *handler.Response
}

// WithAuth подключает auth-модуль к telegram ingress.
func WithAuth(api modulrauth.AuthAPI, cfg AuthConfig) IngressOption {
	return func(target *ingressConfig) {
		if target == nil || api == nil {
			return
		}
		target.auth = api
		target.authCfg = normalizeTelegramAuthConfig(cfg)
	}
}

func buildIngressConfig(opts ...IngressOption) ingressConfig {
	cfg := ingressConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func registerTelegramAuthCommands(api BotAPI, cfg ingressConfig, defaultScope string) error {
	if cfg.auth == nil {
		return nil
	}

	if err := api.RegisterCommand("login", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		currentScope := activeTelegramScope(req.State, defaultScope)
		loginCtx := modulrauth.WithSessionScope(ctx, currentScope)
		if len(cfg.authCfg.AllowedScopes) != 0 {
			loginCtx = modulrauth.WithAllowedScopes(loginCtx, cfg.authCfg.AllowedScopes)
		}

		token, err := cfg.auth.CreateSession(loginCtx, strconv.FormatInt(req.UserID, 10), authRolesForTelegramUser(req.UserID, cfg.authCfg))
		if err != nil {
			return nil, err
		}
		ref := modulrauth.SessionReference(token)
		sess, err := cfg.auth.ValidateSessionReference(ctx, ref)
		if err != nil {
			return nil, err
		}

		return &handler.Response{
			Text:      formatTelegramAuthSession("Auth session активна", sess),
			ParseMode: "Markdown",
			NextState: state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, currentScope), ref),
		}, nil
	}); err != nil {
		return err
	}

	if err := api.RegisterCommand("whoami", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		currentScope := activeTelegramScope(req.State, defaultScope)
		ref := state.AuthSessionReference(req.State)
		if ref == "" {
			return &handler.Response{
				Text:      fmt.Sprintf("Auth session не найдена.\nТекущий scope: `%s`\nИспользуй `/login`.", currentScope),
				ParseMode: "Markdown",
				NextState: state.SetActiveScope(state.Session{}, currentScope),
			}, nil
		}

		sess, err := cfg.auth.ValidateSessionReference(ctx, ref)
		if err != nil {
			return &handler.Response{
				Text:      fmt.Sprintf("Auth session недействительна.\nТекущий scope: `%s`\nИспользуй `/login`.", currentScope),
				ParseMode: "Markdown",
				NextState: state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, currentScope), ""),
			}, nil
		}

		text := formatTelegramAuthSession("Auth session найдена", sess)
		if !modulrauth.ScopeAllowed(sess, currentScope) {
			text += fmt.Sprintf("\nТекущий scope `%s` вне allowed_scopes этой сессии.", currentScope)
		}
		return &handler.Response{
			Text:      text,
			ParseMode: "Markdown",
			NextState: state.SetActiveScope(state.Session{}, currentScope),
		}, nil
	}); err != nil {
		return err
	}

	if err := api.RegisterCommand("logout", func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		currentScope := activeTelegramScope(req.State, defaultScope)
		ref := state.AuthSessionReference(req.State)
		if ref != "" {
			if err := cfg.auth.RevokeSessionReference(ctx, ref); err != nil {
				return nil, err
			}
		}
		return &handler.Response{
			Text:      fmt.Sprintf("Auth session очищена.\nТекущий scope: `%s`", currentScope),
			ParseMode: "Markdown",
			NextState: state.SetAuthSessionReference(state.SetActiveScope(state.Session{}, currentScope), ""),
		}, nil
	}); err != nil {
		return err
	}

	return nil
}

func authorizeTelegramIngress(ctx context.Context, req *handler.Request, currentScope string, cfg ingressConfig) (ingressAuthResult, error) {
	if cfg.auth == nil {
		return ingressAuthResult{}, nil
	}

	ref := state.AuthSessionReference(req.State)
	if ref == "" {
		if !cfg.authCfg.Required {
			return ingressAuthResult{}, nil
		}
		return ingressAuthResult{
			Response: &handler.Response{
				Text:      fmt.Sprintf("Для scope `%s` нужна auth session.\nИспользуй `/login`.", currentScope),
				ParseMode: "Markdown",
			},
		}, nil
	}

	sess, err := cfg.auth.ValidateSessionReference(ctx, ref)
	if err != nil {
		clearState := state.SetAuthSessionReference(state.Session{}, "")
		if !cfg.authCfg.Required {
			return ingressAuthResult{NextState: clearState}, nil
		}
		return ingressAuthResult{
			NextState: clearState,
			Response: &handler.Response{
				Text:      fmt.Sprintf("Auth session истекла или недействительна для scope `%s`.\nИспользуй `/login`.", currentScope),
				ParseMode: "Markdown",
			},
		}, nil
	}

	if !modulrauth.ScopeAllowed(sess, currentScope) {
		if !cfg.authCfg.Required {
			return ingressAuthResult{}, nil
		}
		return ingressAuthResult{
			Response: &handler.Response{
				Text:      fmt.Sprintf("Текущий scope `%s` не входит в allowed_scopes auth session.\nСмени scope или обнови сессию через `/login`.", currentScope),
				ParseMode: "Markdown",
			},
		}, nil
	}

	return ingressAuthResult{Session: sess}, nil
}

func normalizeTelegramAuthConfig(cfg AuthConfig) AuthConfig {
	out := AuthConfig{
		Required:      cfg.Required,
		AdminUserIDs:  dedupeInt64(cfg.AdminUserIDs),
		AllowedScopes: dedupeStrings(cfg.AllowedScopes),
	}
	return out
}

func authRolesForTelegramUser(userID int64, cfg AuthConfig) []modulrauth.Role {
	for _, item := range cfg.AdminUserIDs {
		if item == userID {
			return []modulrauth.Role{modulrauth.RoleAdmin}
		}
	}
	return []modulrauth.Role{modulrauth.RoleUser}
}

func formatTelegramAuthSession(prefix string, sess *modulrauth.Session) string {
	if sess == nil {
		return prefix
	}
	roles := make([]string, 0, len(sess.Roles))
	for _, role := range sess.Roles {
		roles = append(roles, string(role))
	}
	return fmt.Sprintf(
		"%s\nUser: `%s`\nScope: `%s`\nAllowed scopes: `%s`\nRoles: `%s`\nExpires: `%s`",
		prefix,
		sess.UserID,
		sess.Scope,
		strings.Join(sess.AllowedScopes, "`, `"),
		strings.Join(roles, "`, `"),
		sess.ExpiresAt.UTC().Format(time.RFC3339),
	)
}

func dedupeInt64(items []int64) []int64 {
	if len(items) == 0 {
		return nil
	}
	out := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func mergeTelegramState(base, overlay state.Session) state.Session {
	out := state.Session{
		Key:     overlay.Key,
		Payload: cloneTelegramPayload(base.Payload),
	}
	if out.Key == "" {
		out.Key = base.Key
	}
	if len(overlay.Payload) == 0 {
		return out
	}
	if out.Payload == nil {
		out.Payload = make(map[string]interface{}, len(overlay.Payload))
	}
	for k, v := range overlay.Payload {
		out.Payload[k] = v
	}
	return out
}

func cloneTelegramPayload(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
