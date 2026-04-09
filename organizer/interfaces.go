package organizer

import "context"

// Storage интерфейс хранилища (легко заменить на databases.DB позже)
type Storage interface {
	Save(entity interface{}) error
	GetByID(entityType EntityType, id string, out interface{}) error
	List(entityType EntityType, out interface{}) error
	Delete(entityType EntityType, id string) error
}

// AIOrchestrator интерфейс для будущего ИИ-ядра
type AIOrchestrator interface {
	AnalyzeContext(ctx context.Context, event Event) []Suggestion
	ApplySuggestion(ctx context.Context, s Suggestion) error
}

// Suggestion предложение от ИИ или правил автоматизации
type Suggestion struct {
	TargetModule string
	Action       string
	Payload      interface{}
	Confidence   float64 // 0.0 - 1.0
}

// OrganizerAPI публичный контракт для внешних сервисов (Telegram, Web, CLI)
type OrganizerAPI interface {
	// Модули
	Todo() TodoService
	Calendar() CalendarService
	Notes() NoteService
	Contacts() ContactService

	// Управление
	PublishEvent(evt Event)
	RegisterAI(ai AIOrchestrator)
	Start(ctx context.Context) error
	Stop() error
}

// Сервисные интерфейсы (каждый модуль реализует свой)
type TodoService interface {
	Create(ctx context.Context, t *Todo) error
	Get(id string) (*Todo, error)
	List() ([]Todo, error)
	Update(id string, t *Todo) error
	Delete(id string) error
}

type CalendarService interface {
	Create(ctx context.Context, e *CalendarEvent) error
	Get(id string) (*CalendarEvent, error)
	List() ([]CalendarEvent, error)
}

type NoteService interface {
	Create(ctx context.Context, n *Note) error
	Get(id string) (*Note, error)
	List() ([]Note, error)
}

type ContactService interface {
	Create(ctx context.Context, c *Contact) error
	Get(id string) (*Contact, error)
	List() ([]Contact, error)
}
