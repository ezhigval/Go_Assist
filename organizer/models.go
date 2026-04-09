package organizer

import "time"

// EntityType определяет тип сущности органайзера
type EntityType string

const (
	TypeTodo    EntityType = "todo"
	TypeEvent   EntityType = "calendar_event"
	TypeNote    EntityType = "note"
	TypeContact EntityType = "contact"
)

// BaseEntity базовая структура для всех сущностей (готово для ИИ)
type BaseEntity struct {
	ID        string                 `json:"id"`
	Scope     string                 `json:"scope,omitempty"` // жизненный контекст LEGO: personal, family, work, …
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Meta      map[string]interface{} `json:"meta,omitempty"` // Гибкое поле для ИИ-контекста, тегов, связей
}

// Todo задача
type Todo struct {
	BaseEntity
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Completed   bool      `json:"completed"`
	Priority    int       `json:"priority"` // 1-5
}

// CalendarEvent событие календаря
type CalendarEvent struct {
	BaseEntity
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
}

// Note заметка
type Note struct {
	BaseEntity
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// Contact контакт
type Contact struct {
	BaseEntity
	Name    string   `json:"name"`
	Phone   string   `json:"phone"`
	Email   string   `json:"email"`
	Company string   `json:"company"`
	Notes   []string `json:"notes"` // ID связанных заметок
}
