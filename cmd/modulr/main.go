// Сборочный щит: подключаешь модули в порядке внедрения, ядро шины одно.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"modulr/app"
)

func main() {
	ctx := context.Background()
	rt := app.NewRuntime()
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("runtime start: %v", err)
	}
	defer func() {
		if err := rt.Stop(context.Background()); err != nil {
			log.Printf("runtime stop: %v", err)
		}
	}()

	traceID, err := rt.HandleMessage(ctx, app.InboundMessage{
		ChatID:   42,
		UserID:   1001,
		Username: "demo",
		Text:     getEnv("MODULR_DEMO_TEXT", "напоминание купить молоко после работы"),
		Scope:    getEnv("MODULR_DEMO_SCOPE", "personal"),
		Source:   "telegram",
	})
	if err != nil {
		log.Fatalf("handle message: %v", err)
	}

	time.Sleep(400 * time.Millisecond)
	stats, err := rt.Orchestrator().GetStats(ctx)
	if err != nil {
		log.Fatalf("runtime stats: %v", err)
	}
	metricsSnapshot := rt.Metrics().Snapshot(5)
	keys, err := rt.Store().ListPrefix(ctx, "tracker:check:")
	if err != nil {
		log.Fatalf("storage list: %v", err)
	}

	log.Printf(
		"runtime trace=%s checklist_items=%d errors=%d events=%v scope_counts=%v traces=%v",
		traceID,
		len(keys),
		stats.ErrorCount,
		stats.EventCounts,
		metricsSnapshot.ScopeCounts,
		metricsSnapshot.Traces,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
