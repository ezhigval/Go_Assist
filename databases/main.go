package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"databases"
)

func main() {
	cfg := databases.LoadConfig()

	// Инициализация в фоновом контексте
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := databases.InitDB(ctx, cancel)
	cancel() // освобождаем контекст инициализации

	if err != nil {
		log.Fatalf("❌ InitDB failed: %v", err)
	}

	// Graceful shutdown
	termCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.Start(termCtx); err != nil {
		log.Fatalf("❌ DB Start failed: %v", err)
	}

	// 🧩 Пример использования API из другого микросервиса
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("🔍 Тестовый запрос к API БД...")

		user, _ := db.GetOrCreateUser(termCtx, 123456789, "test_user")
		log.Printf("✅ User: %+v", user)

		chat, _ := db.GetOrCreateChat(termCtx, 987654321, "Test Group", "group")
		log.Printf("✅ Chat: %+v", chat)

		_ = db.LogAction(termCtx, user.TgID, "user_login", map[string]interface{}{"ip": "127.0.0.1"})

		stats, _ := db.GetStats(termCtx)
		log.Printf("📊 Stats: %+v", stats)
	}()

	log.Println("🚀 Database service is running. Press Ctrl+C to exit.")
	<-termCtx.Done()

	log.Println("🛑 Shutting down...")
	db.Stop()
}
