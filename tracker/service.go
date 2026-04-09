package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"modulr/events"
)

// Service реализует TrackerAPI.
type Service struct {
	storage events.Storage
	bus     *events.EventBus
	mu      sync.RWMutex
}

// NewService создаёт трекер.
func NewService(st events.Storage, bus *events.EventBus) *Service {
	return &Service{storage: st, bus: bus}
}

// RegisterSubscriptions кросс-связи: календарь, email, заметки → трекер и напоминания.
func (s *Service) RegisterSubscriptions(bus *events.EventBus) {
	if bus == nil {
		return
	}
	s.bus = bus
	bus.Subscribe(events.V1CalendarMeetingCreated, s.onCalendarMeeting)
	bus.Subscribe(events.V1EmailReceived, s.onEmailReceived)
	bus.Subscribe(events.V1NoteCreated, s.onNoteCreated)
}

func (s *Service) onCalendarMeeting(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	title, _ := m["title"].(string)
	seg := events.ParseSegmentFromAny(m["context"])
	p := &Plan{
		Title:       fmt.Sprintf("План по встрече: %s", title),
		Description: "Авто из календаря",
		Milestones: []Milestone{
			{
				ID:    fmt.Sprintf("ms_%d", time.Now().UnixNano()),
				Title: "Подготовка к встрече",
				DueAt: time.Now().Add(24 * time.Hour),
				Done:  false,
			},
		},
	}
	p.Context = seg
	if p.Context == "" {
		p.Context = events.SegmentWork
	}
	p.Tags = []string{"calendar_bridge", "meeting"}
	// STUB: Meeting decomposition requires agenda text analysis via orchestrator/AI to generate []Milestone for Plan without hardcoded single stage.
	if err := s.CreatePlan(ctx, p); err != nil {
		log.Printf("tracker: calendar→plan: %v", err)
	}
}

func (s *Service) onEmailReceived(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	subj, _ := m["subject"].(string)
	if !invoiceLike(subj) {
		return
	}
	item := &CheckListItem{
		Title:    fmt.Sprintf("Оплатить: %s", subj),
		DueAt:    time.Now().Add(72 * time.Hour),
		Done:     false,
		Source:   "email_invoice",
		LinkedID: fmt.Sprint(m["id"]),
	}
	item.Context = events.ParseSegmentFromAny(m["context"])
	if item.Context == "" {
		item.Context = events.DefaultSegment()
	}
	item.Tags = []string{"deadline", "finance_bridge"}
	if err := s.AddChecklistItem(ctx, item); err != nil {
		log.Printf("tracker: email→checklist: %v", err)
	}
}

var buyRe = regexp.MustCompile(`(?i)купить\s+\S+`)

func (s *Service) onNoteCreated(evt events.Event) {
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	content, _ := m["content"].(string)
	seg := events.ParseSegmentFromAny(m["context"])
	if !buyRe.MatchString(content) {
		return
	}
	// STUB: Geo for v1.reminder.on_route requires maps/geocoding API integration with lat/lon and geofence_km from user profile and scope.
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name: events.V1ReminderOnRoute,
			Payload: map[string]any{
				"hint":        strings.TrimSpace(buyRe.FindString(content)),
				"context":     string(seg),
				"lat":         0.0,
				"lon":         0.0,
				"note_id":     m["id"],
				"geofence_km": 2.0,
			},
			Source:  "tracker",
			TraceID: evt.TraceID,
		})
	}
}

func invoiceLike(subj string) bool {
	s := strings.ToLower(subj)
	return strings.Contains(s, "инвойс") || strings.Contains(s, "invoice") ||
		strings.Contains(s, "оплат") || strings.Contains(s, "payment")
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

// CreatePlan сохраняет план и шлёт событие plan.created.
func (s *Service) CreatePlan(ctx context.Context, p *Plan) error {
	if p == nil {
		return fmt.Errorf("tracker: nil plan")
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("plan_%d", time.Now().UnixNano())
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	if err := s.storage.PutJSON(ctx, "tracker:plan:"+p.ID, p); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventPlanCreated,
			Payload: p,
			Source:  "tracker",
			Context: map[string]any{"segment": string(p.Context)},
		})
	}
	return nil
}

// ReachMilestone отмечает этап и публикует v1.tracker.milestone.reached (глобальное имя на шине).
func (s *Service) ReachMilestone(ctx context.Context, planID, milestoneID string) error {
	var p Plan
	if err := s.storage.GetJSON(ctx, "tracker:plan:"+planID, &p); err != nil {
		return err
	}
	var hit *Milestone
	for i := range p.Milestones {
		if p.Milestones[i].ID == milestoneID {
			p.Milestones[i].Done = true
			p.Milestones[i].ReachedAt = time.Now()
			hit = &p.Milestones[i]
			break
		}
	}
	if hit == nil {
		return fmt.Errorf("tracker: milestone not found")
	}
	p.UpdatedAt = time.Now()
	if err := s.storage.PutJSON(ctx, "tracker:plan:"+planID, &p); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name: events.V1TrackerMilestoneReached,
			Payload: map[string]any{
				"plan_id":   planID,
				"title":     hit.Title,
				"context":   string(p.Context),
				"tags":      p.Tags,
				"milestone": milestoneID,
			},
			Source: "tracker",
		})
	}
	return nil
}

// LogHabit фиксирует выполнение привычки.
func (s *Service) LogHabit(ctx context.Context, habitID string, at time.Time) error {
	var h Habit
	if err := s.storage.GetJSON(ctx, "tracker:habit:"+habitID, &h); err != nil {
		return err
	}
	if at.IsZero() {
		at = time.Now()
	}
	if isSameDay(h.LastLogged, at) {
		h.Streak++
	} else if h.LastLogged.IsZero() || at.Sub(h.LastLogged) > 36*time.Hour {
		h.Streak = 1
	} else {
		h.Streak++
	}
	h.LastLogged = at
	h.UpdatedAt = time.Now()
	if err := s.storage.PutJSON(ctx, "tracker:habit:"+habitID, &h); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name: EventHabitLogged,
			Payload: map[string]any{
				"habit_id": habitID,
				"streak":   h.Streak,
				"at":       at.Format(time.RFC3339),
			},
			Source: "tracker",
		})
	}
	return nil
}

func isSameDay(a, b time.Time) bool {
	if a.IsZero() {
		return false
	}
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// AddChecklistItem добавляет пункт.
func (s *Service) AddChecklistItem(ctx context.Context, item *CheckListItem) error {
	if item == nil {
		return fmt.Errorf("tracker: nil checklist item")
	}
	if item.ID == "" {
		item.ID = fmt.Sprintf("chk_%d", time.Now().UnixNano())
	}
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	return s.storage.PutJSON(ctx, "tracker:check:"+item.ID, item)
}

// ListPlans возвращает планы (фильтр по сегменту).
func (s *Service) ListPlans(ctx context.Context, segment string) ([]Plan, error) {
	keys, err := s.storage.ListPrefix(ctx, "tracker:plan:")
	if err != nil {
		return nil, err
	}
	var out []Plan
	for _, k := range keys {
		var p Plan
		if err := s.storage.GetJSON(ctx, k, &p); err != nil {
			continue
		}
		if segment != "" && string(p.Context) != segment {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
