package organizer

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// EventName тип события
type EventName string

const (
	EventTodoCreated     EventName = "todo.created"
	EventTodoCompleted   EventName = "todo.completed"
	EventEventCreated    EventName = "calendar.created"
	EventNoteSaved       EventName = "note.saved"
	EventContactCreated  EventName = "contact.created"
	EventAIContextUpdate EventName = "ai.context_update"
)

// Event структура события
type Event struct {
	Name      EventName
	Payload   interface{}
	Source    string
	Timestamp time.Time
	Context   map[string]interface{}
}

// EventBus асинхронная шина событий (Pub/Sub)
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventName][]func(Event)
}

// NewEventBus создаёт шину
func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[EventName][]func(Event))}
}

// Subscribe регистрирует обработчик события
func (bus *EventBus) Subscribe(name EventName, fn func(Event)) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers[name] = append(bus.handlers[name], fn)
}

// Publish асинхронно отправляет событие всем подписчикам
func (bus *EventBus) Publish(evt Event) {
	evt.Timestamp = time.Now()
	if evt.Context == nil {
		evt.Context = make(map[string]interface{})
	}

	bus.mu.RLock()
	handlers := make([]func(Event), len(bus.handlers[evt.Name]))
	copy(handlers, bus.handlers[evt.Name])
	bus.mu.RUnlock()

	// Асинхронный вызов, чтобы не блокировать основной поток
	for _, h := range handlers {
		go func(fn func(Event)) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("⚠️ Event handler panic [%s]: %v", evt.Name, r)
				}
			}()
			fn(evt)
		}(h)
	}
}
