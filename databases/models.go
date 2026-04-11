package databases

import "time"

// User представляет пользователя Telegram
type User struct {
	ID        int64     `json:"id"`
	TgID      int64     `json:"tg_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Chat представляет чат (личный, групповой, канал)
type Chat struct {
	ID        int64     `json:"id"`
	TgID      int64     `json:"tg_id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // private, group, supergroup, channel
	CreatedAt time.Time `json:"created_at"`
}

// Session хранит состояние диалога и временные данные
type Session struct {
	ID        int64                  `json:"id"`
	ChatID    int64                  `json:"chat_id"`
	State     string                 `json:"state"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// EventJournalEntry хранит trace-связанный журнал transport/runtime событий.
type EventJournalEntry struct {
	ID        int64                  `json:"id"`
	TraceID   string                 `json:"trace_id"`
	ChatID    int64                  `json:"chat_id"`
	Scope     string                 `json:"scope"`
	EventName string                 `json:"event_name"`
	Status    string                 `json:"status"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// StatsSummary агрегированная статистика
type StatsSummary struct {
	TotalUsers   int64 `json:"total_users"`
	TotalChats   int64 `json:"total_chats"`
	TotalActions int64 `json:"total_actions"`
}
