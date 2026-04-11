package events

import (
	"context"
	"log"
	"sync"
)

// Name идентификатор события на шине (формат v1.{module}.{action}).
type Name string

// Event универсальное сообщение шины.
type Event struct {
	Name    Name           `json:"name"`
	Payload any            `json:"payload"`
	Context map[string]any `json:"context,omitempty"`
	ChatID  int64          `json:"chat_id"`
	// Scope доменный контекст изоляции: personal | family | business.
	Scope string   `json:"scope"`
	Tags  []string `json:"tags,omitempty"`
}

// Handler обработчик события (всегда с context.Context).
type Handler func(ctx context.Context, e Event)

// EventBus контракт pub/sub без привязки к конкретной реализации транспорта.
type EventBus interface {
	// Publish доставляет событие всем подписчикам (асинхронно; ошибка — отмена контекста или сбой шины).
	Publish(ctx context.Context, e Event) error
	// Subscribe регистрирует обработчик на точное имя события.
	Subscribe(n Name, h Handler)
}

// MemoryBus потокобезопасная in-memory реализация EventBus для ядра и тестов.
type MemoryBus struct {
	mu   sync.RWMutex
	subs map[Name][]Handler
	all  []Handler
}

// NewMemoryBus создаёт шину.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{subs: make(map[Name][]Handler)}
}

// Publish ставит доставку в фоне: не блокирует вызывающего, внутри — WaitGroup по обработчикам.
func (b *MemoryBus) Publish(ctx context.Context, e Event) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	b.mu.RLock()
	handlers := append([]Handler(nil), b.subs[e.Name]...)
	handlers = append(handlers, b.all...)
	b.mu.RUnlock()

	go func() {
		var wg sync.WaitGroup
		for _, fn := range handlers {
			if fn == nil {
				continue
			}
			h := fn
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
				}
				defer func() {
					if r := recover(); r != nil {
						log.Printf("core/events: handler panic [%s]: %v", e.Name, r)
					}
				}()
				h(ctx, e)
			}()
		}
		wg.Wait()
	}()
	return nil
}

// Subscribe добавляет обработчик.
func (b *MemoryBus) Subscribe(n Name, h Handler) {
	if h == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[n] = append(b.subs[n], h)
}

// SubscribeAll регистрирует пассивный слушатель всех событий шины.
func (b *MemoryBus) SubscribeAll(h Handler) {
	if h == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.all = append(b.all, h)
}
