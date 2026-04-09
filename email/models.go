package email

import (
	"time"

	"modulr/events"
)

// EmailMessage нормализованное письмо (IMAP/SMTP — за пределами пакета).
type EmailMessage struct {
	events.EntityBase
	From        string          `json:"from"`
	To          []string        `json:"to"`
	Subject     string          `json:"subject"`
	BodyText    string          `json:"body_text"`
	ReceivedAt  time.Time       `json:"received_at"`
	MessageID   string          `json:"message_id"`
	Attachments []Attachment    `json:"attachments"`
	Flags       map[string]bool `json:"flags"`
}

// Attachment вложение.
type Attachment struct {
	Filename string `json:"filename"`
	MIME     string `json:"mime"`
	Size     int64  `json:"size"`
	Ref      string `json:"ref"`
}

// Rule правило фильтрации/маршрутизации входящих.
type Rule struct {
	events.EntityBase
	Name       string         `json:"name"`
	MatchField string         `json:"match_field"` // subject, from, body
	Pattern    string         `json:"pattern"`     // regexp string
	Action     string         `json:"action"`      // tag, forward_stub, mark_priority
	ActionArgs map[string]any `json:"action_args"`
	Priority   int            `json:"priority"`
	Active     bool           `json:"active"`
}
