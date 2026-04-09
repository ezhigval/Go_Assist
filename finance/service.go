package finance

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

// Service реализует FinanceAPI; кросс-модульные реакции только через RegisterSubscriptions + Publish.
type Service struct {
	storage events.Storage
	bus     *events.EventBus

	mu        sync.RWMutex
	budgets   map[string]int64 // ключ: segment|category -> лимит (minor)
	spent     map[string]int64 // накопленный расход по категории в сегменте
	templates []categoryRule
}

type categoryRule struct {
	pattern  *regexp.Regexp
	tag      string
	category string
}

// NewService создаёт сервис; storage/bus обязательны для публикации и персистенции.
func NewService(st events.Storage, bus *events.EventBus) *Service {
	s := &Service{
		storage: st,
		bus:     bus,
		budgets: make(map[string]int64),
		spent:   make(map[string]int64),
		templates: []categoryRule{
			{pattern: regexp.MustCompile(`(?i)еда|ресторан|coffee`), tag: "food", category: "food"},
			{pattern: regexp.MustCompile(`(?i)такси|uber|транспорт`), tag: "mobility", category: "transport"},
			{pattern: regexp.MustCompile(`(?i)офис|работа|saas`), tag: "work_expense", category: "work"},
		},
	}
	return s
}

func budgetKey(seg events.Segment, cat string) string {
	return string(seg) + "|" + cat
}

// RegisterSubscriptions подписывает finance на внешние события (без импорта других модулей).
func (s *Service) RegisterSubscriptions(bus *events.EventBus) {
	if bus == nil {
		return
	}
	s.bus = bus
	bus.Subscribe(events.V1CalendarMeetingCreated, s.onCalendarMeetingCreated)
	bus.Subscribe(events.V1EmailReceived, s.onEmailReceived)
	bus.Subscribe(events.V1TrackerMilestoneReached, s.onMilestoneForPaidCourse)
	bus.Subscribe(EventTransactionCreated, s.onTransactionCreatedRecalc)
}

func (s *Service) onTransactionCreatedRecalc(evt events.Event) {
	ctx := context.Background()
	t, ok := decodeTransaction(evt.Payload)
	if !ok || t == nil {
		return
	}
	s.recheckBudget(ctx, t)
}

func (s *Service) onCalendarMeetingCreated(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadMap(evt.Payload)
	if !ok {
		return
	}
	seg := events.ParseSegmentFromAny(m["context"])
	if !events.IsCareerScope(seg) {
		return
	}
	title, _ := m["title"].(string)
	tx := &Transaction{
		Type:         TransactionExpense,
		AmountMinor:  0,
		Currency:     "RUB",
		Category:     "work",
		Counterparty: "meeting_auto",
		Memo:         fmt.Sprintf("Встреча: %s", title),
	}
	tx.Context = seg
	if tx.Context == "" {
		tx.Context = events.SegmentWork
	}
	tx.Tags = []string{"work_expense", "auto_from_calendar"}
	tx.EntityBase.CreatedAt = time.Now()
	tx.EntityBase.UpdatedAt = time.Now()
	// STUB: Meeting cost estimation requires AI integration for expense calculation and confidence threshold validation.
	if err := s.CreateTransaction(ctx, tx); err != nil {
		log.Printf("finance: calendar→transaction: %v", err)
	}
}

func (s *Service) onMilestoneForPaidCourse(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadMap(evt.Payload)
	if !ok {
		return
	}
	tags := stringSlice(m["tags"])
	if !containsTag(tags, "paid_course") {
		return
	}
	title, _ := m["title"].(string)
	tx := &Transaction{
		Type:         TransactionExpense,
		AmountMinor:  0,
		Currency:     "RUB",
		Category:     "education",
		Counterparty: "course",
		Memo:         fmt.Sprintf("Этап курса: %s", title),
	}
	tx.Context = events.ParseSegmentFromAny(m["context"])
	if tx.Context == "" {
		tx.Context = events.DefaultSegment()
	}
	tx.Tags = append(tags, "milestone_finance_link")
	tx.EntityBase.CreatedAt = time.Now()
	tx.EntityBase.UpdatedAt = time.Now()
	// STUB: Course fee deduction requires contract/invoice matching service integration to set AmountMinor before CreateTransaction.
	if err := s.CreateTransaction(ctx, tx); err != nil {
		log.Printf("finance: milestone→transaction: %v", err)
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if strings.EqualFold(strings.TrimPrefix(t, "#"), want) {
			return true
		}
	}
	return false
}

func stringSlice(v any) []string {
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

func (s *Service) onEmailReceived(evt events.Event) {
	ctx := context.Background()
	m, ok := payloadMap(evt.Payload)
	if !ok {
		return
	}
	subj, _ := m["subject"].(string)
	if !invoiceLike(subj) {
		return
	}
	seg := events.ParseSegmentFromAny(m["context"])
	if seg == "" {
		seg = events.DefaultSegment()
	}
	tx := &Transaction{
		Type:         TransactionExpense,
		AmountMinor:  0,
		Currency:     "RUB",
		Category:     "invoice",
		Counterparty: "email_parser",
		Memo:         subj,
	}
	tx.Context = seg
	tx.Tags = []string{"invoice", "email_ingest"}
	tx.EntityBase.CreatedAt = time.Now()
	tx.EntityBase.UpdatedAt = time.Now()
	// STUB: Invoice parsing from email requires LLM/regex entity extraction on BodyText with scope validation before transaction parameters.
	if err := s.CreateTransaction(ctx, tx); err != nil {
		log.Printf("finance: email→transaction: %v", err)
	}
}

func invoiceLike(subj string) bool {
	s := strings.ToLower(subj)
	return strings.Contains(s, "инвойс") || strings.Contains(s, "invoice") ||
		strings.Contains(s, "оплат") || strings.Contains(s, "payment")
}

func payloadMap(p any) (map[string]any, bool) {
	if m, ok := p.(map[string]any); ok {
		return m, true
	}
	b, ok := p.([]byte)
	if ok {
		var out map[string]any
		if err := json.Unmarshal(b, &out); err == nil {
			return out, true
		}
	}
	return nil, false
}

func decodeTransaction(p any) (*Transaction, bool) {
	m, ok := payloadMap(p)
	if !ok {
		return nil, false
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}
	var t Transaction
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// CreateTransaction сохраняет операцию и публикует событие.
func (s *Service) CreateTransaction(ctx context.Context, t *Transaction) error {
	if t == nil {
		return fmt.Errorf("finance: nil transaction")
	}
	s.applyAutoCategory(t)
	if t.ID == "" {
		t.ID = fmt.Sprintf("txn_%d", time.Now().UnixNano())
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	key := "finance:transaction:" + t.ID
	if err := s.storage.PutJSON(ctx, key, t); err != nil {
		return err
	}
	s.bumpSpent(t)
	if s.bus != nil {
		s.bus.Publish(events.Event{
			Name:    EventTransactionCreated,
			Payload: t,
			Source:  "finance",
			Context: map[string]any{"segment": string(t.Context)},
		})
	}
	s.recheckBudget(ctx, t)
	return nil
}

func (s *Service) recheckBudget(_ context.Context, t *Transaction) {
	if t.Type != TransactionExpense {
		return
	}
	k := budgetKey(t.Context, t.Category)
	s.mu.RLock()
	limit := s.budgets[k]
	sp := s.spent[k]
	s.mu.RUnlock()
	if limit <= 0 {
		return
	}
	if sp > limit && s.bus != nil {
		s.bus.Publish(events.Event{
			Name: EventBudgetExceeded,
			Payload: map[string]any{
				"segment":  string(t.Context),
				"category": t.Category,
				"spent":    sp,
				"limit":    limit,
			},
			Source: "finance",
		})
	}
}

func (s *Service) bumpSpent(t *Transaction) {
	if t.Type != TransactionExpense {
		return
	}
	k := budgetKey(t.Context, t.Category)
	s.mu.Lock()
	s.spent[k] += t.AmountMinor
	s.mu.Unlock()
}

// applyAutoCategory эвристики по тегам/тексту; ИИ — отдельно.
func (s *Service) applyAutoCategory(t *Transaction) {
	if t.Category != "" {
		return
	}
	hay := strings.ToLower(strings.Join(t.Tags, " ") + " " + t.Memo)
	for _, r := range s.templates {
		if strings.Contains(hay, r.tag) || r.pattern.MatchString(hay) {
			t.Category = r.category
			return
		}
	}
	// STUB: Semantic categorization requires classifier service call on Memo+Tags with fallback to current categoryRule regex patterns.
	t.Category = "uncategorized"
}

// GetTransaction возвращает операцию по id.
func (s *Service) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var t Transaction
	key := "finance:transaction:" + id
	if err := s.storage.GetJSON(ctx, key, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTransactions фильтрует по сегменту (пустая строка — все).
func (s *Service) ListTransactions(ctx context.Context, segment string) ([]Transaction, error) {
	keys, err := s.storage.ListPrefix(ctx, "finance:transaction:")
	if err != nil {
		return nil, err
	}
	var out []Transaction
	for _, k := range keys {
		var t Transaction
		if err := s.storage.GetJSON(ctx, k, &t); err != nil {
			continue
		}
		if segment != "" && string(t.Context) != segment {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// UpsertSubscription сохраняет подписку.
func (s *Service) UpsertSubscription(ctx context.Context, sub *Subscription) error {
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("sub_%d", time.Now().UnixNano())
	}
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	return s.storage.PutJSON(ctx, "finance:subscription:"+sub.ID, sub)
}

// ListSubscriptions возвращает все подписки.
func (s *Service) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	keys, err := s.storage.ListPrefix(ctx, "finance:subscription:")
	if err != nil {
		return nil, err
	}
	var out []Subscription
	for _, k := range keys {
		var sub Subscription
		if err := s.storage.GetJSON(ctx, k, &sub); err != nil {
			continue
		}
		out = append(out, sub)
	}
	return out, nil
}

// UpsertCredit сохраняет кредит.
func (s *Service) UpsertCredit(ctx context.Context, c *Credit) error {
	if c.ID == "" {
		c.ID = fmt.Sprintf("cr_%d", time.Now().UnixNano())
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	return s.storage.PutJSON(ctx, "finance:credit:"+c.ID, c)
}

// UpsertInvestment сохраняет инвестицию.
func (s *Service) UpsertInvestment(ctx context.Context, inv *Investment) error {
	if inv.ID == "" {
		inv.ID = fmt.Sprintf("inv_%d", time.Now().UnixNano())
	}
	now := time.Now()
	inv.CreatedAt = now
	inv.UpdatedAt = now
	return s.storage.PutJSON(ctx, "finance:investment:"+inv.ID, inv)
}

// BalanceBySegment сумма income - expense по сегменту.
func (s *Service) BalanceBySegment(ctx context.Context, segment string) (int64, error) {
	list, err := s.ListTransactions(ctx, segment)
	if err != nil {
		return 0, err
	}
	var bal int64
	for _, t := range list {
		switch t.Type {
		case TransactionIncome:
			bal += t.AmountMinor
		case TransactionExpense:
			bal -= t.AmountMinor
		}
	}
	return bal, nil
}

// SetBudgetLimit задаёт лимит расходов по категории в сегменте.
func (s *Service) SetBudgetLimit(ctx context.Context, segment, category string, limitMinor int64) error {
	_ = ctx
	if segment == "" || category == "" {
		return fmt.Errorf("finance: empty segment/category")
	}
	s.mu.Lock()
	s.budgets[budgetKey(events.Segment(segment), category)] = limitMinor
	s.mu.Unlock()
	return nil
}
