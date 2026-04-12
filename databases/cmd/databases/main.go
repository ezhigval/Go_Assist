package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"databases"
)

func main() {
	command, args := parseCommand(os.Args[1:])

	switch command {
	case "serve":
		runServe()
	case "up":
		runUp()
	case "down":
		runDown(args)
	case "status":
		runStatus()
	case "rls-status":
		runRLSStatus(args)
	case "app-role-sql":
		runAppRoleSQL(args)
	case "stats":
		runStats(args)
	case "journal":
		runJournal(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		log.Fatalf("❌ Unknown command %q\n\n%s", command, usageText())
	}
}

func parseCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "up", nil
	}
	return args[0], args[1:]
}

func runServe() {
	cfg := databases.LoadConfig()

	termCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db := mustInitDB(termCtx, cfg)
	defer closeDB(db)

	if err := db.Start(termCtx); err != nil {
		log.Fatalf("❌ DB Start failed: %v", err)
	}

	log.Println("🚀 databases serve: connection is healthy, waiting for shutdown signal")
	<-termCtx.Done()
	log.Println("🛑 databases serve: shutting down")
}

func runUp() {
	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	if err := db.ApplyMigrations(ctx); err != nil {
		log.Fatalf("❌ migrate up failed: %v", err)
	}
}

func runDown(args []string) {
	flags := flag.NewFlagSet("down", flag.ExitOnError)
	steps := flags.Int("steps", 1, "number of migrations to rollback")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: go run ./cmd/databases down [-steps=1]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(args)

	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	if err := db.RollbackMigrations(ctx, *steps); err != nil {
		log.Fatalf("❌ migrate down failed: %v", err)
	}
}

func runStatus() {
	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	statuses, err := db.ListMigrationStatus(ctx)
	if err != nil {
		log.Fatalf("❌ migration status failed: %v", err)
	}

	fmt.Printf("%-8s %-12s %-24s %s\n", "VERSION", "STATE", "APPLIED_AT", "NAME")
	for _, status := range statuses {
		appliedAt := "-"
		if status.AppliedAt != nil {
			appliedAt = status.AppliedAt.UTC().Format(time.RFC3339)
		}
		fmt.Printf("%06d %-12s %-24s %s\n", status.Version, status.State(), appliedAt, status.Name)
	}
}

func runJournal(args []string) {
	flags := flag.NewFlagSet("journal", flag.ExitOnError)
	traceID := flags.String("trace", "", "trace_id for replay")
	chatID := flags.Int64("chat", 0, "chat_id for reverse-chronological journal view")
	scope := flags.String("scope", "", "base scope for scoped replay/read (required unless -all-scopes)")
	allowScopes := flags.String("allow-scopes", "", "comma-separated extra scopes allowed for this read")
	allScopes := flags.Bool("all-scopes", false, "bypass scope filter for admin/deploy tooling")
	limit := flags.Int("limit", 50, "max number of journal entries")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: go run ./cmd/databases journal (-trace=<trace_id> | -chat=<chat_id>) [-scope=personal] [-allow-scopes=business,travel] [-limit=50] [-all-scopes]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(args)

	if (*traceID == "" && *chatID == 0) || (*traceID != "" && *chatID != 0) {
		log.Fatalf("❌ journal requires exactly one selector: -trace or -chat\n\n%s", usageText())
	}

	filter, err := buildScopeFilter(*scope, *allowScopes, *allScopes)
	if err != nil {
		log.Fatalf("❌ build journal filter failed: %v", err)
	}

	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	var entries []databases.EventJournalEntry
	switch {
	case *traceID != "":
		entries, err = db.ListJournalEventsByTraceScoped(ctx, *traceID, filter, *limit)
	default:
		entries, err = db.ListJournalEventsByChatScoped(ctx, *chatID, filter, *limit)
	}
	if err != nil {
		log.Fatalf("❌ journal query failed: %v", err)
	}

	if len(entries) == 0 {
		log.Println("journal: no rows")
		return
	}

	for _, entry := range entries {
		fmt.Printf("%s trace=%s chat=%d scope=%s status=%s source=%s event=%s\n",
			entry.CreatedAt.UTC().Format(time.RFC3339),
			entry.TraceID,
			entry.ChatID,
			entry.Scope,
			entry.Status,
			entry.Source,
			entry.EventName,
		)
		if payload := mustMarshalJSON(entry.Payload); payload != "" {
			fmt.Printf("  payload=%s\n", payload)
		}
		if metadata := mustMarshalJSON(entry.Metadata); metadata != "" {
			fmt.Printf("  metadata=%s\n", metadata)
		}
	}
}

func runRLSStatus(args []string) {
	flags := flag.NewFlagSet("rls-status", flag.ExitOnError)
	requireEffective := flags.Bool("require-effective", false, "exit with non-zero code when storage RLS is not effective")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: go run ./cmd/databases rls-status [-require-effective]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(args)

	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	status, err := db.InspectStorageRLS(ctx)
	if err != nil {
		log.Fatalf("❌ storage RLS status failed: %v", err)
	}

	fmt.Printf("current_user=%s\n", status.CurrentUser)
	fmt.Printf("effective=%t\n", status.Effective())
	fmt.Printf("role_superuser=%t\n", status.RoleSuperuser)
	fmt.Printf("role_bypass_rls=%t\n", status.RoleBypassRLS)
	printTableRLSStatus(status.Journal, status.RoleSuperuser, status.RoleBypassRLS)
	printTableRLSStatus(status.Stats, status.RoleSuperuser, status.RoleBypassRLS)
	printTableRLSStatus(status.Sessions, status.RoleSuperuser, status.RoleBypassRLS)
	printTableRLSStatus(status.AuthSessions, status.RoleSuperuser, status.RoleBypassRLS)
	for _, warning := range status.Warnings() {
		fmt.Printf("warning=%s\n", warning)
	}
	if err := databases.EnforceStorageRLS(status, *requireEffective); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

func runAppRoleSQL(args []string) {
	flags := flag.NewFlagSet("app-role-sql", flag.ExitOnError)
	role := flags.String("role", "modulr_app", "login role for application connections")
	schema := flags.String("schema", "public", "target schema for grants/default privileges")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: go run ./cmd/databases app-role-sql [-role=modulr_app] [-schema=public]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(args)

	cfg := databases.LoadConfig()
	sql, err := databases.BuildAppRoleBootstrapSQL(*role, cfg.Name, *schema)
	if err != nil {
		log.Fatalf("❌ build app role SQL failed: %v", err)
	}
	fmt.Print(sql)
}

func runStats(args []string) {
	flags := flag.NewFlagSet("stats", flag.ExitOnError)
	scope := flags.String("scope", "", "base scope for scoped stats (required unless -all-scopes)")
	allowScopes := flags.String("allow-scopes", "", "comma-separated extra scopes allowed for this read")
	allScopes := flags.Bool("all-scopes", false, "bypass scope filter for admin/deploy tooling")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: go run ./cmd/databases stats [-scope=personal] [-allow-scopes=business,travel] [-all-scopes]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(args)

	filter, err := buildScopeFilter(*scope, *allowScopes, *allScopes)
	if err != nil {
		log.Fatalf("❌ build stats filter failed: %v", err)
	}

	cfg := databases.LoadConfig()
	cfg.AutoMigrate = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := mustInitDB(ctx, cfg)
	defer closeDB(db)

	summary, err := db.GetActionStatsScoped(ctx, filter)
	if err != nil {
		log.Fatalf("❌ stats query failed: %v", err)
	}

	fmt.Printf("total_actions=%d\n", summary.TotalActions)
	for _, key := range sortedCountKeys(summary.ScopeCounts) {
		fmt.Printf("scope_count[%s]=%d\n", key, summary.ScopeCounts[key])
	}
	for _, key := range sortedCountKeys(summary.ActionCounts) {
		fmt.Printf("action_count[%s]=%d\n", key, summary.ActionCounts[key])
	}
}

func mustInitDB(ctx context.Context, cfg databases.Config) *databases.DB {
	db, err := databases.InitDB(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ InitDB failed: %v", err)
	}
	return db
}

func closeDB(db *databases.DB) {
	if err := db.Stop(); err != nil {
		log.Printf("⚠️ DB Stop failed: %v", err)
	}
}

func printUsage() {
	fmt.Print(usageText())
}

func usageText() string {
	return `databases CLI

Commands:
  up                 apply all pending migrations (default)
  down [-steps=N]    rollback the last N migrations
  status             print migration state
  rls-status         inspect effective storage RLS state for current DB role
  app-role-sql       print bootstrap SQL for a non-superuser application role
  stats              print scoped stats action aggregates
  journal            print scope-aware event_journal entries by trace_id or chat_id
  serve              connect, optionally auto-migrate, and wait for shutdown
  help               print this help

Environment:
  DB_AUTO_MIGRATE=true  affects only "serve" and application startup via databases.InitDB
`
}

func buildScopeFilter(baseScope, allowScopesCSV string, allScopes bool) (databases.JournalScopeFilter, error) {
	if allScopes {
		if strings.TrimSpace(baseScope) != "" || strings.TrimSpace(allowScopesCSV) != "" {
			return databases.JournalScopeFilter{}, fmt.Errorf("-all-scopes cannot be combined with -scope or -allow-scopes")
		}
		return databases.FullJournalScopeFilter(), nil
	}
	if strings.TrimSpace(baseScope) == "" {
		return databases.JournalScopeFilter{}, fmt.Errorf("-scope is required unless -all-scopes is set")
	}

	metadata := map[string]any{}
	if scopes := parseCSVScopes(allowScopesCSV); len(scopes) > 0 {
		metadata["allowed_scopes"] = scopes
	}
	return databases.NewJournalScopeFilter(baseScope, nil, metadata)
}

func buildJournalFilter(baseScope, allowScopesCSV string, allScopes bool) (databases.JournalScopeFilter, error) {
	return buildScopeFilter(baseScope, allowScopesCSV, allScopes)
}

func parseCSVScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(strings.ToLower(part))
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func mustMarshalJSON(v map[string]interface{}) string {
	if len(v) == 0 {
		return ""
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("{\"marshal_error\":%q}", err.Error())
	}
	return string(raw)
}

func sortedCountKeys(m map[string]int64) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func printTableRLSStatus(status databases.TableRLSStatus, roleSuperuser, roleBypassRLS bool) {
	fmt.Printf("table[%s].effective=%t\n", status.TableName, status.Effective(roleSuperuser, roleBypassRLS))
	fmt.Printf("table[%s].rls_enabled=%t\n", status.TableName, status.TableRLSEnabled)
	fmt.Printf("table[%s].rls_forced=%t\n", status.TableName, status.TableRLSForced)
	fmt.Printf("table[%s].select_policy=%t\n", status.TableName, status.SelectPolicy)
	fmt.Printf("table[%s].insert_policy=%t\n", status.TableName, status.InsertPolicy)
}
