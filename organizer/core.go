package organizer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"organizer/calendar"
	"organizer/contacts"
	"organizer/notes"
	"organizer/todo"
)

// Organizer ядро-оркестратор
type Organizer struct {
	cfg        Config
	bus        *EventBus
	storage    Storage
	ai         AIOrchestrator
	todoSvc    *todo.Service
	calSvc     *calendar.Service
	noteSvc    *notes.Service
	contactSvc *contacts.Service
	mu         sync.Mutex
}

// NewOrganizer создаёт и инициализирует ядро
func NewOrganizer(cfg Config, storage Storage) *Organizer {
	org := &Organizer{
		cfg:     cfg,
		bus:     NewEventBus(),
		storage: storage,
	}

	// Инициализация модулей
	org.todoSvc = todo.NewService(storage)
	org.calSvc = calendar.NewService(storage, org.bus)
	org.noteSvc = notes.NewService(storage, org.bus)
	org.contactSvc = contacts.NewService(storage)

	// Регистрация кросс-модульных связей (правила автоматизации)
	org.registerIntegrations()

	return org
}

// registerIntegrations настраивает автоматические взаимодействия.
// STUB: Dynamic rules require replacing hard Subscribe with AIOrchestrator.AnalyzeContext event flow with scope/tags validation before bus publishing.
func (org *Organizer) registerIntegrations() {
	if !org.cfg.EnableCrossLinks {
		return
	}

	// 1. Календарь → Todo: При создании встречи автоматически создаётся задача
	org.bus.Subscribe(EventEventCreated, func(evt Event) {
		e, ok := evt.Payload.(*CalendarEvent)
		if !ok {
			return
		}

		t := &Todo{
			Title:       fmt.Sprintf("📅 Встреча: %s", e.Title),
			Description: e.Description,
			DueDate:     e.StartTime,
			Priority:    3,
		}
		t.CreatedAt = time.Now()
		t.UpdatedAt = time.Now()
		t.Meta = map[string]interface{}{"source": "calendar", "event_id": e.ID}

		if err := org.todoSvc.Create(context.Background(), t); err != nil {
			log.Printf("⚠️ Auto-todo creation failed: %v", err)
		} else {
			log.Printf("🔗 Auto-linked calendar event to todo: %s", e.Title)
		}
	})

	// 2. Заметки → Контакты: Сканирование текста на телефоны/почты
	org.bus.Subscribe(EventNoteSaved, func(evt Event) {
		n, ok := evt.Payload.(*Note)
		if !ok {
			return
		}

		// Простой парсер (в будущем вынесется в notes/parser.go или ИИ)
		phones := org.noteSvc.ExtractPhones(n.Content)
		emails := org.noteSvc.ExtractEmails(n.Content)

		for _, phone := range phones {
			c := &Contact{
				Name:  fmt.Sprintf("Контакт из заметки: %s", n.Title),
				Phone: phone,
			}
			c.CreatedAt = time.Now()
			c.UpdatedAt = time.Now()
			c.Meta = map[string]interface{}{"source": "note", "note_id": n.ID}
			org.bus.Publish(Event{
				Name:    EventContactCreated,
				Payload: c,
				Source:  "notes.auto_extract",
				Context: map[string]interface{}{"auto": true},
			})
		}
		for _, email := range emails {
			c := &Contact{Email: email}
			c.CreatedAt = time.Now()
			c.UpdatedAt = time.Now()
			c.Meta = map[string]interface{}{"source": "note", "note_id": n.ID}
			org.bus.Publish(Event{
				Name:    EventContactCreated,
				Payload: c,
				Source:  "notes.auto_extract",
				Context: map[string]interface{}{"auto": true},
			})
		}
	})

	// 3. ИИ-хук: перехват всех событий для анализа
	org.bus.Subscribe(EventAIContextUpdate, func(evt Event) {
		if org.ai != nil {
			suggestions := org.ai.AnalyzeContext(context.Background(), evt)
			for _, s := range suggestions {
				// STUB: Suggestion application requires org.ai.ApplySuggestion(ctx, s) with TargetModule/Action whitelist validation and rejection logging.
				log.Printf("🤖 AI Suggestion: %s -> %s (%.2f)", s.TargetModule, s.Action, s.Confidence)
			}
		}
	})
}

// Реализация OrganizerAPI
func (org *Organizer) Todo() TodoService         { return org.todoSvc }
func (org *Organizer) Calendar() CalendarService { return org.calSvc }
func (org *Organizer) Notes() NoteService        { return org.noteSvc }
func (org *Organizer) Contacts() ContactService  { return org.contactSvc }

func (org *Organizer) PublishEvent(evt Event) { org.bus.Publish(evt) }

func (org *Organizer) RegisterAI(ai AIOrchestrator) { org.ai = ai }

func (org *Organizer) Start(ctx context.Context) error {
	log.Println("🧩 Organizer core initialized")
	return nil
}

func (org *Organizer) Stop() error {
	log.Println("🛑 Organizer core stopped")
	return nil
}
