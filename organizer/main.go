package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"organizer"
)

func main() {
	ctx := context.Background()
	api := organizer.QuickStart(ctx)
	defer api.Stop()

	log.Println("🚀 Organizer Multitool Demo")
	log.Println("===========================")

	// 1. Создаём встречу в календаре
	log.Println("\n📅 1. Creating calendar event...")
	_ = api.Calendar().Create(ctx, &organizer.CalendarEvent{
		Title:       "Встреча с командой",
		Description: "Обсуждение архитектуры монорепозитория",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(25 * time.Hour),
	})

	// 2. Ждём асинхронную обработку события (Календарь → Todo)
	time.Sleep(500 * time.Millisecond)
	todos, _ := api.Todo().List()
	log.Printf("✅ Todos after calendar event: %d", len(todos))
	for _, t := range todos {
		log.Printf("   - %s (Due: %s)", t.Title, t.DueDate.Format("15:04"))
	}

	// 3. Сохраняем заметку с телефоном
	log.Println("\n📝 2. Saving note with phone...")
	_ = api.Notes().Create(ctx, &organizer.Note{
		Title:   "Важные контакты",
		Content: "Звонить Ивану: +79991234567 или ivan@example.com",
		Tags:    []string{"work", "urgent"},
	})

	time.Sleep(500 * time.Millisecond)
	contacts, _ := api.Contacts().List()
	log.Printf("✅ Contacts auto-extracted: %d", len(contacts))
	for _, c := range contacts {
		log.Printf("   - %s (%s)", c.Phone, c.Email)
	}

	// 4. Демонстрация ИИ-хука
	log.Println("\n🤖 3. Simulating AI context update...")
	api.PublishEvent(organizer.Event{
		Name:    organizer.EventAIContextUpdate,
		Payload: "User mentioned 'deadline tomorrow' in todo",
		Source:  "ai.simulation",
	})

	fmt.Println("\n✅ Organizer demo finished. All modules interconnected.")
}
