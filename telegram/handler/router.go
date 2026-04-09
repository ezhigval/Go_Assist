package handler

import (
	"context"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram/message"
	"telegram/state"
)

// Router маршрутизирует обновления к зарегистрированным обработчикам
type Router struct {
	mu           sync.RWMutex
	commands     map[string]HandlerFunc
	callbacks    map[string]HandlerFunc
	states       map[string]HandlerFunc
	textHandlers []struct {
		prefix string
		fn     HandlerFunc
	}
	store       state.Store
	middlewares []MiddlewareFunc
}

// MiddlewareFunc — цепочка обработчиков до/после вызова
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// NewRouter создаёт маршрутизатор
func NewRouter(store state.Store) *Router {
	return &Router{
		commands:  make(map[string]HandlerFunc),
		callbacks: make(map[string]HandlerFunc),
		states:    make(map[string]HandlerFunc),
		store:     store,
	}
}

// Use добавляет middleware
func (r *Router) Use(mw ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, mw...)
}

// RegisterCommand регистрирует обработчик команды
func (r *Router) RegisterCommand(cmd string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd] = h
}

// RegisterText регистрирует обработчик текста (по префиксу)
func (r *Router) RegisterText(prefix string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.textHandlers = append(r.textHandlers, struct {
		prefix string
		fn     HandlerFunc
	}{prefix, h})
}

// RegisterCallback регистрирует обработчик inline-кнопок
func (r *Router) RegisterCallback(prefix string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks[prefix] = h
}

// RegisterState регистрирует обработчик по состоянию диалога
func (r *Router) RegisterState(stateKey string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[stateKey] = h
}

// Handle обрабатывает входящее обновление
func (r *Router) Handle(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil && update.CallbackQuery == nil {
		return nil
	}

	req := &Request{}
	var fn HandlerFunc

	if update.CallbackQuery != nil {
		q := update.CallbackQuery
		req.ChatID = q.Message.Chat.ID
		req.UserID = q.From.ID
		req.Username = q.From.UserName
		req.Callback = q.Data
		req.State = r.store.Get(ctx, req.ChatID)
		fn = r.findCallbackHandler(q.Data)
	} else {
		msg := update.Message
		req.ChatID = msg.Chat.ID
		req.UserID = msg.From.ID
		req.Username = msg.From.UserName
		req.Text = msg.Text
		req.Command = msg.Command()
		req.State = r.store.Get(ctx, req.ChatID)

		if req.Command != "" {
			fn = r.commands[req.Command]
		} else if req.State.Key != "" {
			fn = r.states[req.State.Key]
		} else {
			fn = r.findTextHandler(msg.Text)
		}
	}

	if fn == nil {
		return nil // Нет обработчика → игнорируем
	}

	// Применяем middleware
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		fn = r.middlewares[i](fn)
	}

	resp, err := fn(ctx, req)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	// Сохраняем новое состояние
	if resp.NextState.Key != "" {
		r.store.Set(ctx, req.ChatID, resp.NextState)
	} else {
		r.store.Clear(ctx, req.ChatID)
	}

	// Возвращаем Response для дальнейшей отправки (обрабатывается в bot.go)
	return nil
}

func (r *Router) findCallbackHandler(data string) HandlerFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for prefix, fn := range r.callbacks {
		if strings.HasPrefix(data, prefix) {
			return fn
		}
	}
	return nil
}

func (r *Router) findTextHandler(text string) HandlerFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.textHandlers {
		if strings.HasPrefix(text, h.prefix) {
			return h.fn
		}
	}
	return nil
}
