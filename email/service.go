package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"modulr/events"
)

// Service реализует EmailAPI.
type Service struct {
	storage events.Storage
	bus     *events.EventBus

	mu    sync.RWMutex
	rules []Rule
}

// NewService создаёт email-модуль.
func NewService(st events.Storage, bus *events.EventBus) *Service {
	return &Service{storage: st, bus: bus, rules: nil}
}

// RegisterSubscriptions: календарь → напоминание по email (заглушка IMAP/SMTP).
func (s *Service) RegisterSubscriptions(bus *events.EventBus) {
	if bus == nil {
		return
	}
	s.bus = bus
	bus.Subscribe(events.V1CalendarMeetingCreated, s.onCalendarMeeting)
}

func (s *Service) onCalendarMeeting(evt events.Event) {
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	title, _ := m["title"].(string)
	start := ""
	switch v := m["start_time"].(type) {
	case string:
		start = v
	case time.Time:
		start = v.Format(time.RFC3339)
	default:
		start = fmt.Sprint(v)
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name: EventActionRequired,
			Payload: map[string]any{
				"kind":    "send_reminder_email",
				"title":   title,
				"start":   start,
				"trace":   evt.TraceID,
				"context": m["context"],
			},
			Source: "email",
		})
	}
}

func payloadToMap(p any) (map[string]any, bool) {
	if m, ok := p.(map[string]any); ok {
		return m, true
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

// IngestIncoming парсит/нормализует входящее и публикует v1.email.received.
func (s *Service) IngestIncoming(ctx context.Context, msg *EmailMessage) error {
	if msg == nil {
		return fmt.Errorf("email: nil message")
	}
	s.applyRules(msg)
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("em_%d", time.Now().UnixNano())
	}
	now := time.Now()
	msg.ReceivedAt = now
	msg.CreatedAt = now
	msg.UpdatedAt = now
	if err := s.storage.PutJSON(ctx, "email:message:"+msg.ID, msg); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    events.V1EmailReceived,
			Payload: msgToMap(msg),
			Source:  "email",
			Context: map[string]any{"segment": string(msg.Context)},
		})
	}
	return nil
}

func msgToMap(msg *EmailMessage) map[string]any {
	return map[string]any{
		"id":         msg.ID,
		"subject":    msg.Subject,
		"body":       msg.BodyText,
		"context":    string(msg.Context),
		"tags":       msg.Tags,
		"from":       msg.From,
		"message_id": msg.MessageID,
	}
}

func (s *Service) applyRules(msg *EmailMessage) {
	s.mu.RLock()
	rules := append([]Rule(nil), s.rules...)
	s.mu.RUnlock()
	for _, r := range rules {
		if !r.Active {
			continue
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			continue
		}
		var field string
		switch r.MatchField {
		case "subject":
			field = msg.Subject
		case "from":
			field = msg.From
		default:
			field = msg.BodyText
		}
		if re.MatchString(field) {
			msg.Tags = append(msg.Tags, r.Name)
		}
	}
	// STUB: Incoming classification requires AI classifier call with only metadata stored in Tags/Message.Flags, no raw body in long-term cache.
}

// SendOutgoing отправка (SMTP за пределами); публикует v1.email.sent.
func (s *Service) SendOutgoing(ctx context.Context, msg *EmailMessage) error {
	if msg == nil {
		return fmt.Errorf("email: nil message")
	}
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("out_%d", time.Now().UnixNano())
	}
	now := time.Now()
	msg.UpdatedAt = now
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	if err := s.storage.PutJSON(ctx, "email:out:"+msg.ID, msg); err != nil {
		return err
	}
	// STUB: Reply drafts require optional LLM call on thread_id history with user confirmation before SendOutgoing.
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventSent,
			Payload: msgToMap(msg),
			Source:  "email",
		})
	}
	return nil
}

// UpsertRule сохраняет правило.
func (s *Service) UpsertRule(ctx context.Context, r *Rule) error {
	if r.ID == "" {
		r.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if err := s.storage.PutJSON(ctx, "email:rule:"+r.ID, r); err != nil {
		return err
	}
	s.mu.Lock()
	s.rules = append(s.rules, *r)
	s.mu.Unlock()
	return nil
}

// ListRules возвращает правила из памяти и хранилища (объединение упрощённо — из стора).
func (s *Service) ListRules(ctx context.Context) ([]Rule, error) {
	keys, err := s.storage.ListPrefix(ctx, "email:rule:")
	if err != nil {
		return nil, err
	}
	var out []Rule
	for _, k := range keys {
		var r Rule
		if err := s.storage.GetJSON(ctx, k, &r); err != nil {
			log.Printf("email: list rule %s: %v", k, err)
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
