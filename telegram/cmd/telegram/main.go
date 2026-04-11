package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"modulr/app"
	"telegram"
	"telegram/examples"
)

func main() {
	cfg := telegram.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	persistence, err := initPersistence(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Init persistence failed: %v", err)
	}
	defer func() {
		if err := persistence.close(); err != nil {
			log.Printf("⚠️ Persistence stop failed: %v", err)
		}
	}()

	rt := app.NewRuntime(persistence.runtimeOpts...)
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("❌ Runtime start failed: %v", err)
	}
	defer func() {
		if err := rt.Stop(context.Background()); err != nil {
			log.Printf("⚠️ Runtime stop failed: %v", err)
		}
	}()

	bot, err := telegram.InitBot(cfg, persistence.store)
	if err != nil {
		log.Fatalf("❌ InitBot failed: %v", err)
	}

	examples.RegisterStartHandler(bot)
	if err := telegram.RegisterModulrIngress(bot, rt, cfg.DefaultScope, cfg.RuntimeTimeout); err != nil {
		log.Fatalf("❌ RegisterModulrIngress failed: %v", err)
	}

	log.Println("🚀 Bot is running. Press Ctrl+C to exit.")
	if err := bot.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("🛑 Bot crashed: %v", err)
	}
}
