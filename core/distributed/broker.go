package distributed

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrTopicRequired означает, что брокеру не передали topic.
	ErrTopicRequired = errors.New("distributed: topic required")
	// ErrGroupRequired означает, что подписка не указала consumer group.
	ErrGroupRequired = errors.New("distributed: group required")
	// ErrHandlerRequired означает, что подписка вызвана без handler.
	ErrHandlerRequired = errors.New("distributed: handler required")
	// ErrDuplicateConsumer означает повторную регистрацию consumer id в той же группе.
	ErrDuplicateConsumer = errors.New("distributed: duplicate consumer in group")
)

var consumerSeq atomic.Uint64

// Envelope — транспортная форма события для распределённой шины.
// Контракт намеренно близок к core/events.Event, но не привязан к конкретной реализации bus.
type Envelope struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Payload     any            `json:"payload"`
	Context     map[string]any `json:"context,omitempty"`
	ChatID      int64          `json:"chat_id,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	TraceID     string         `json:"trace_id,omitempty"`
	PublishedAt time.Time      `json:"published_at"`
}

// Delivery описывает конкретную доставку consumer group.
type Delivery struct {
	Topic    string   `json:"topic"`
	Group    string   `json:"group"`
	Consumer string   `json:"consumer"`
	Offset   uint64   `json:"offset"`
	Attempt  int      `json:"attempt"`
	Envelope Envelope `json:"envelope"`
}

// Handler обрабатывает сообщение consumer group.
type Handler func(context.Context, Delivery) error

// Subscription позволяет снять подписку без перезапуска брокера.
type Subscription interface {
	Close() error
}

// Broker задаёт минимальный контракт для распределённой шины v2.0.
type Broker interface {
	Publish(ctx context.Context, topic string, envelope Envelope) error
	SubscribeGroup(ctx context.Context, topic, group, consumer string, handler Handler) (Subscription, error)
	Stats(topic string) TopicStats
}

// TopicStats — диагностический срез по topic.
type TopicStats struct {
	Topic           string       `json:"topic"`
	Published       uint64       `json:"published"`
	GroupCount      int          `json:"group_count"`
	SubscriberCount int          `json:"subscriber_count"`
	Groups          []GroupStats `json:"groups,omitempty"`
}

// GroupStats — диагностический срез по consumer group.
type GroupStats struct {
	Group       string `json:"group"`
	Subscribers int    `json:"subscribers"`
	Delivered   uint64 `json:"delivered"`
	Failed      uint64 `json:"failed"`
}

// MemoryBroker — локальная реализация broker-контракта для тестов и single-process стендов.
// Семантика: каждый publish доставляется по одному consumer в каждой group (round-robin).
type MemoryBroker struct {
	mu     sync.RWMutex
	topics map[string]*topicState
	seq    uint64
}

type topicState struct {
	published uint64
	groups    map[string]*groupState
}

type groupState struct {
	subscribers []*subscriberState
	next        int
	delivered   uint64
	failed      uint64
}

type subscriberState struct {
	id      string
	handler Handler
}

type memorySubscription struct {
	broker   *MemoryBroker
	topic    string
	group    string
	consumer string
	once     sync.Once
}

type deliveryTarget struct {
	group    string
	consumer string
	handler  Handler
}

// NewMemoryBroker создаёт in-memory broker с consumer-group семантикой.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		topics: make(map[string]*topicState),
	}
}

// Publish публикует envelope в topic и синхронно прогоняет consumer-group handlers.
// Ошибки consumer не ломают publish path, а отражаются только в статистике группы.
func (b *MemoryBroker) Publish(ctx context.Context, topic string, envelope Envelope) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ErrTopicRequired
	}

	prepared := cloneEnvelope(envelope)
	if prepared.ID == "" {
		prepared.ID = fmt.Sprintf("%s-%d", strings.ReplaceAll(topic, "/", "_"), atomic.AddUint64(&b.seq, 1))
	}
	if prepared.PublishedAt.IsZero() {
		prepared.PublishedAt = time.Now().UTC()
	}

	var (
		targets []deliveryTarget
		offset  uint64
	)

	b.mu.Lock()
	state := b.ensureTopicLocked(topic)
	state.published++
	offset = state.published
	targets = make([]deliveryTarget, 0, len(state.groups))
	for groupName, group := range state.groups {
		sub := group.nextSubscriber()
		if sub == nil {
			continue
		}
		targets = append(targets, deliveryTarget{
			group:    groupName,
			consumer: sub.id,
			handler:  sub.handler,
		})
	}
	b.mu.Unlock()

	for _, target := range targets {
		err := target.handler(ctx, Delivery{
			Topic:    topic,
			Group:    target.group,
			Consumer: target.consumer,
			Offset:   offset,
			Attempt:  1,
			Envelope: cloneEnvelope(prepared),
		})
		b.recordDelivery(topic, target.group, err)
	}
	return nil
}

// SubscribeGroup подписывает handler на topic/group. Внутри одной группы сообщения распределяются round-robin.
func (b *MemoryBroker) SubscribeGroup(ctx context.Context, topic, group, consumer string, handler Handler) (Subscription, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return nil, ErrGroupRequired
	}
	if handler == nil {
		return nil, ErrHandlerRequired
	}
	consumer = strings.TrimSpace(consumer)
	if consumer == "" {
		consumer = fmt.Sprintf("consumer-%d", consumerSeq.Add(1))
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	topicState := b.ensureTopicLocked(topic)
	groupEntry := topicState.groups[group]
	if groupEntry == nil {
		groupEntry = &groupState{}
		topicState.groups[group] = groupEntry
	}
	for _, sub := range groupEntry.subscribers {
		if sub.id == consumer {
			return nil, ErrDuplicateConsumer
		}
	}
	groupEntry.subscribers = append(groupEntry.subscribers, &subscriberState{
		id:      consumer,
		handler: handler,
	})
	return &memorySubscription{
		broker:   b,
		topic:    topic,
		group:    group,
		consumer: consumer,
	}, nil
}

// Stats возвращает диагностический snapshot topic.
func (b *MemoryBroker) Stats(topic string) TopicStats {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return TopicStats{}
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	state := b.topics[topic]
	if state == nil {
		return TopicStats{Topic: topic}
	}

	groups := make([]GroupStats, 0, len(state.groups))
	subscriberCount := 0
	groupNames := make([]string, 0, len(state.groups))
	for groupName := range state.groups {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)
	for _, groupName := range groupNames {
		group := state.groups[groupName]
		subscriberCount += len(group.subscribers)
		groups = append(groups, GroupStats{
			Group:       groupName,
			Subscribers: len(group.subscribers),
			Delivered:   group.delivered,
			Failed:      group.failed,
		})
	}

	return TopicStats{
		Topic:           topic,
		Published:       state.published,
		GroupCount:      len(state.groups),
		SubscriberCount: subscriberCount,
		Groups:          groups,
	}
}

func (b *MemoryBroker) ensureTopicLocked(topic string) *topicState {
	state := b.topics[topic]
	if state == nil {
		state = &topicState{
			groups: make(map[string]*groupState),
		}
		b.topics[topic] = state
	}
	return state
}

func (b *MemoryBroker) recordDelivery(topic, group string, handlerErr error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.topics[topic]
	if state == nil {
		return
	}
	groupState := state.groups[group]
	if groupState == nil {
		return
	}
	groupState.delivered++
	if handlerErr != nil {
		groupState.failed++
	}
}

func (g *groupState) nextSubscriber() *subscriberState {
	if len(g.subscribers) == 0 {
		return nil
	}
	index := g.next % len(g.subscribers)
	g.next = (index + 1) % len(g.subscribers)
	return g.subscribers[index]
}

func (s *memorySubscription) Close() error {
	s.once.Do(func() {
		s.broker.removeSubscriber(s.topic, s.group, s.consumer)
	})
	return nil
}

func (b *MemoryBroker) removeSubscriber(topic, group, consumer string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.topics[topic]
	if state == nil {
		return
	}
	groupState := state.groups[group]
	if groupState == nil {
		return
	}

	for i, sub := range groupState.subscribers {
		if sub.id != consumer {
			continue
		}
		groupState.subscribers = append(groupState.subscribers[:i], groupState.subscribers[i+1:]...)
		if len(groupState.subscribers) == 0 {
			delete(state.groups, group)
			return
		}
		if groupState.next >= len(groupState.subscribers) {
			groupState.next = 0
		}
		return
	}
}

func cloneEnvelope(in Envelope) Envelope {
	out := in
	out.Context = cloneMap(in.Context)
	out.Tags = cloneStrings(in.Tags)
	if payload, ok := in.Payload.(map[string]any); ok {
		out.Payload = cloneMap(payload)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
