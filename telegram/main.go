package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"telegram"
	"telegram/examples"
	"telegram/state"
)

func main() {
	cfg := telegram.LoadConfig()
	store := state.NewMemoryStore() // В продакшене заменить на RedisStore

	bot, err := telegram.InitBot(cfg, store)
	if err != nil {
		log.Fatalf("❌ InitBot failed: %v", err)
	}

	// 🧩 Регистрация модулей (плагинов)
	examples.RegisterStartHandler(bot)
	// calculator.Register(bot)
	// catalog.Register(bot)
	// admin.Register(bot)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 Bot is running. Press Ctrl+C to exit.")
	if err := bot.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("🛑 Bot crashed: %v", err)
	}
}
