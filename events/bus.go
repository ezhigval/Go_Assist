package events

import (
	"log"
	"strings"
	"sync"
	"time"
)

// Handler обработчик события.
type Handler func(Event)

// Bus асинхронная шина: точные имена, суффикс (*.due), глобальные подписчики, dead-letter.
type Bus struct {
	mu          sync.RWMutex
	exact       map[Name][]Handler
	suffix      []suffixSub
	all         []Handler
	deadLetter  chan Event
	deadLetterN int
}

type suffixSub struct {
	suffix string
	fn     Handler
}

// NewBus создаёт шину. deadLetterBuf > 0 включает неблокирующий буфер dead-letter при панике в хендлере.
func NewBus(deadLetterBuf int) *Bus {
	b := &Bus{exact: make(map[Name][]Handler)}
	if deadLetterBuf > 0 {
		b.deadLetter = make(chan Event, deadLetterBuf)
		go b.drainDeadLetter()
	}
	return b
}

func (b *Bus) drainDeadLetter() {
	for evt := range b.deadLetter {
		log.Printf("dead-letter [%s] source=%s trace=%s", evt.Name, evt.Source, evt.TraceID)
		b.mu.Lock()
		b.deadLetterN++
		b.mu.Unlock()
	}
}

// DeadLetterCount для метрик/диагностики.
func (b *Bus) DeadLetterCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.deadLetterN
}

// Subscribe подписка на точное имя.
func (b *Bus) Subscribe(name Name, fn Handler) {
	if fn == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.exact[name] = append(b.exact[name], fn)
}

// SubscribeSuffix подписка на суффикс имени (например ".due" для *.due).
func (b *Bus) SubscribeSuffix(suffix string, fn Handler) {
	if fn == nil || suffix == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.suffix = append(b.suffix, suffixSub{suffix: suffix, fn: fn})
}

// SubscribeAll пассивные слушатели всех событий (metrics, ai context).
func (b *Bus) SubscribeAll(fn Handler) {
	if fn == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.all = append(b.all, fn)
}

// Publish асинхронно доставляет событие; паника в хендлере не роняет шину.
func (b *Bus) Publish(evt Event) {
	evt.Timestamp = time.Now()
	if evt.Context == nil {
		evt.Context = make(map[string]any)
	}

	b.mu.RLock()
	exact := append([]Handler(nil), b.exact[evt.Name]...)
	suffixSubs := append([]suffixSub(nil), b.suffix...)
	all := append([]Handler(nil), b.all...)
	b.mu.RUnlock()

	handlers := make([]Handler, 0, len(exact)+len(suffixSubs)+len(all))
	handlers = append(handlers, exact...)
	for _, s := range suffixSubs {
		if strings.HasSuffix(string(evt.Name), s.suffix) {
			handlers = append(handlers, s.fn)
		}
	}
	handlers = append(handlers, all...)

	for _, h := range handlers {
		fn := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("event handler panic [%s]: %v", evt.Name, r)
					b.enqueueDeadLetter(evt)
				}
			}()
			fn(evt)
		}()
	}
}

func (b *Bus) enqueueDeadLetter(evt Event) {
	if b.deadLetter == nil {
		return
	}
	select {
	case b.deadLetter <- evt:
	default:
		log.Printf("dead-letter buffer full, drop [%s]", evt.Name)
	}
}
