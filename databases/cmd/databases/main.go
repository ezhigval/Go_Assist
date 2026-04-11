package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
  serve              connect, optionally auto-migrate, and wait for shutdown
  help               print this help

Environment:
  DB_AUTO_MIGRATE=true  affects only "serve" and application startup via databases.InitDB
`
}
