package tracker

import (
	"time"

	"modulr/events"
)

// Plan учебный или рабочий план с этапами.
type Plan struct {
	events.EntityBase
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Milestones  []Milestone `json:"milestones"`
}

// Milestone этап плана.
type Milestone struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	DueAt     time.Time `json:"due_at"`
	Done      bool      `json:"done"`
	ReachedAt time.Time `json:"reached_at,omitempty"`
}

// Habit привычка с журналом.
type Habit struct {
	events.EntityBase
	Name       string    `json:"name"`
	Cadence    string    `json:"cadence"` // daily, weekly
	Streak     int       `json:"streak"`
	LastLogged time.Time `json:"last_logged"`
}

// CheckListItem пункт чек-листа (в т.ч. дедлайн из email).
type CheckListItem struct {
	events.EntityBase
	Title    string    `json:"title"`
	DueAt    time.Time `json:"due_at"`
	Done     bool      `json:"done"`
	Source   string    `json:"source"`
	LinkedID string    `json:"linked_id"`
}
