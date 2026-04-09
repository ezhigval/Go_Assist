package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"telegram/handler"
	"telegram/message"
	"telegram/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot — ядро Telegram-бота
type Bot struct {
	api    *tgbotapi.BotAPI
	cfg    Config
	router *handler.Router
	state  state.Store
	server *http.Server
	mu     sync.RWMutex
}

// InitBot инициализирует бота и все зависимости
func InitBot(cfg Config, store state.Store) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("init bot API: %w", err)
	}
	log.Printf("✅ Authorized as @%s", api.Self.UserName)

	bot := &Bot{
		api:    api,
		cfg:    cfg,
		router: handler.NewRouter(store),
		state:  store,
	}

	// Базовые middleware
	bot.router.Use(LoggingMiddleware, RecoverMiddleware)

	return bot, nil
}

// Start запускает бота в выбранном режиме
func (b *Bot) Start(ctx context.Context) error {
	switch strings.ToLower(b.cfg.Mode) {
	case "webhook":
		return b.startWebhook(ctx)
	default:
		return b.startPolling(ctx)
	}
}

func (b *Bot) startPolling(ctx context.Context) error {
	log.Println("🚀 Starting in Polling mode...")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	u.AllowedUpdates = b.cfg.AllowedUpdates

	updates := b.api.GetUpdatesChan(u)
	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Polling stopped by context")
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				continue
			}
			b.processUpdate(ctx, update)
		}
	}
}

func (b *Bot) startWebhook(ctx context.Context) error {
	if b.cfg.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required for webhook mode")
	}
	log.Println("🚀 Starting in Webhook mode...")

	wh := tgbotapi.NewWebhook(b.cfg.WebhookURL)
	wh.AllowedUpdates = b.cfg.AllowedUpdates
	if _, err := b.api.SetWebhook(wh); err != nil {
		return fmt.Errorf("set webhook: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(b.cfg.WebhookPath, b.webhookHandler)

	b.server = &http.Server{
		Addr:    ":" + b.cfg.ServerPort,
		Handler: mux,
	}

	go func() {
		log.Printf("🌐 Webhook server listening on :%s", b.cfg.ServerPort)
		if err := b.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("⚠️ HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Webhook shutdown initiated")
	b.server.Shutdown(context.Background())
	b.api.RemoveWebhook()
	return nil
}

func (b *Bot) webhookHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	go b.processUpdate(r.Context(), update)
}

func (b *Bot) processUpdate(ctx context.Context, update tgbotapi.Update) {
	err := b.router.Handle(ctx, update)
	if err != nil {
		log.Printf("❌ Handler error: %v", err)
		// STUB: Error UX requires SendMessage with message.FormatError(err) and chatID from update, respecting ctx cancellation.
		return
	}
}

// Реализация BotAPI интерфейса
func (b *Bot) RegisterCommand(cmd string, h handler.HandlerFunc) error {
	b.router.RegisterCommand(cmd, h)
	return nil
}
func (b *Bot) RegisterText(pattern string, h handler.HandlerFunc) error {
	b.router.RegisterText(pattern, h)
	return nil
}
func (b *Bot) RegisterCallback(prefix string, h handler.HandlerFunc) error {
	b.router.RegisterCallback(prefix, h)
	return nil
}
func (b *Bot) RegisterState(stateKey string, h handler.HandlerFunc) error {
	b.router.RegisterState(stateKey, h)
	return nil
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, resp *handler.Response) error {
	cfg, err := message.BuildMessageConfig(chatID, resp)
	if err != nil {
		return err
	}
	_, err = b.api.Send(cfg)
	return err
}

func (b *Bot) EditMessage(ctx context.Context, chatID int64, msgID int, resp *handler.Response) error {
	// STUB: EditMessage requires tgbotapi.NewEditMessageText/EditMessageReplyMarkup with msgID and telegram API error handling; no-op implementation for BotAPI compatibility.
	return nil
}

func (b *Bot) GetState(chatID int64) state.Session {
	return b.state.Get(context.Background(), chatID)
}

func (b *Bot) SetState(chatID int64, s state.Session) error {
	return b.state.Set(context.Background(), chatID, s)
}

func (b *Bot) Stop() error {
	if b.server != nil {
		return b.server.Close()
	}
	return nil
}

// Middleware: логирование
func LoggingMiddleware(next handler.HandlerFunc) handler.HandlerFunc {
	return func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		log.Printf("📩 [%s] @%s → %s", req.Command, req.Username, req.Text)
		return next(ctx, req)
	}
}

// Middleware: паник-рекавери
func RecoverMiddleware(next handler.HandlerFunc) handler.HandlerFunc {
	return func(ctx context.Context, req *handler.Request) (*handler.Response, error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Recovered from panic: %v", r)
			}
		}()
		return next(ctx, req)
	}
}
