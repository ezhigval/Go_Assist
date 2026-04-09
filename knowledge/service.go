package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"modulr/events"
)

// Service реализует KnowledgeAPI.
type Service struct {
	storage events.Storage
	bus     *events.EventBus
	mu      sync.RWMutex
}

// NewService создаёт модуль знаний.
func NewService(st events.Storage, bus *events.EventBus) *Service {
	return &Service{storage: st, bus: bus}
}

// RegisterSubscriptions: прогресс трекера → артефакт знаний.
func (s *Service) RegisterSubscriptions(bus *events.EventBus) {
	if bus == nil {
		return
	}
	s.bus = bus
	bus.Subscribe(events.V1TrackerMilestoneReached, s.onMilestoneReached)
	bus.Subscribe(EventQuery, s.onKnowledgeQueryEvent)
}

func (s *Service) onKnowledgeQueryEvent(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	q, _ := m["query"].(string)
	seg := events.Segment(fmt.Sprint(m["context"]))
	if _, err := s.Query(ctx, q, seg); err != nil {
		log.Printf("knowledge: query event: %v", err)
	}
}

func (s *Service) onMilestoneReached(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadToMap(evt.Payload)
	if !ok {
		return
	}
	title, _ := m["title"].(string)
	planID, _ := m["plan_id"].(string)
	a := &Article{
		Title:    fmt.Sprintf("Прогресс: %s", title),
		Body:     fmt.Sprintf("Достигнут этап плана %s", planID),
		Source:   "tracker_bus",
		Topics:   []string{"progress", "milestone"},
		Verified: false,
	}
	a.Context = events.ParseSegmentFromAny(m["context"])
	if a.Context == "" {
		a.Context = events.DefaultSegment()
	}
	a.Tags = append([]string{"auto_milestone"}, tagsFromAny(m["tags"])...)
	// STUB: Milestone summarization requires AIGateway integration to generate Title/Body from plan context and tags without duplicating raw user text in logs.
	if err := s.SaveArticle(ctx, a); err != nil {
		log.Printf("knowledge: milestone→article: %v", err)
	}
}

func tagsFromAny(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range arr {
		s, _ := x.(string)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
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

// SaveArticle сохраняет статью и публикует recommendation/saved.
func (s *Service) SaveArticle(ctx context.Context, a *Article) error {
	if a == nil {
		return fmt.Errorf("knowledge: nil article")
	}
	if a.ID == "" {
		a.ID = fmt.Sprintf("art_%d", time.Now().UnixNano())
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if err := s.storage.PutJSON(ctx, "knowledge:article:"+a.ID, a); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventSaved,
			Payload: a,
			Source:  "knowledge",
		})
		s.recommendRelated(ctx, a)
	}
	return nil
}

func (s *Service) recommendRelated(ctx context.Context, a *Article) {
	keys, err := s.storage.ListPrefix(ctx, "knowledge:article:")
	if err != nil {
		return
	}
	var related []string
	for _, k := range keys {
		var o Article
		if err := s.storage.GetJSON(ctx, k, &o); err != nil {
			continue
		}
		if o.ID == a.ID {
			continue
		}
		for _, t := range a.Topics {
			if containsFold(o.Topics, t) {
				related = append(related, o.ID)
				break
			}
		}
	}
	if len(related) == 0 {
		return
	}
	// STUB: Article recommendations require AI scoring and reading history analysis to rank related_ids before publishing EventRecommendation with scope filtering.
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name: EventRecommendation,
			Payload: map[string]any{
				"base_article_id": a.ID,
				"related_ids":     related,
			},
			Source: "knowledge",
		})
	}
}

func containsFold(list []string, want string) bool {
	for _, x := range list {
		if strings.EqualFold(x, want) {
			return true
		}
	}
	return false
}

// Query поиск по заголовку/телу/тегам.
func (s *Service) Query(ctx context.Context, q string, segment events.Segment) (*QueryResult, error) {
	start := time.Now()
	keys, err := s.storage.ListPrefix(ctx, "knowledge:article:")
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	var hits []Article
	for _, k := range keys {
		var a Article
		if err := s.storage.GetJSON(ctx, k, &a); err != nil {
			continue
		}
		if segment != "" && a.Context != segment {
			continue
		}
		if q == "" {
			hits = append(hits, a)
			continue
		}
		if strings.Contains(strings.ToLower(a.Title+" "+a.Body), q) ||
			tagsContain(a.Tags, q) {
			hits = append(hits, a)
		}
	}
	res := &QueryResult{
		Query:   q,
		Hits:    hits,
		Latency: time.Since(start).Milliseconds(),
	}
	return res, nil
}

func tagsContain(tags []string, q string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// VerifyFact заглушка верификации с эвристикой.
func (s *Service) VerifyFact(ctx context.Context, claim string) (*FactCheck, error) {
	if claim == "" {
		return nil, fmt.Errorf("knowledge: empty claim")
	}
	fc := &FactCheck{
		Claim:    claim,
		Verdict:  "unknown",
		Score:    0.5,
		Evidence: nil,
	}
	fc.ID = fmt.Sprintf("fc_%d", time.Now().UnixNano())
	fc.Context = events.DefaultSegment()
	fc.Tags = []string{"factcheck"}
	fc.CreatedAt = time.Now()
	fc.UpdatedAt = fc.CreatedAt
	// STUB: Fact checking requires claim comparison with Article index and external snippet API to set Verdict/Score/Evidence without PII storage.
	if err := s.storage.PutJSON(ctx, "knowledge:fact:"+fc.ID, fc); err != nil {
		return nil, err
	}
	return fc, nil
}

// UpsertTopicGraph сохраняет узел графа тем.
func (s *Service) UpsertTopicGraph(ctx context.Context, g *TopicGraph) error {
	if g.ID == "" {
		g.ID = fmt.Sprintf("tg_%d", time.Now().UnixNano())
	}
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	return s.storage.PutJSON(ctx, "knowledge:topic:"+g.ID, g)
}
