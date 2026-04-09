package orchestrator

import (
	"sync"
	"time"

	coreevents "modulr/core/events"
)

const maxChatHistory = 50

// HistoryItem запись контекста по чату для последующей подачи в AI.
type HistoryItem struct {
	Time    time.Time `json:"time"`
	Name    string    `json:"name"`
	Scope   string    `json:"scope"`
	Summary string    `json:"summary"`
}

// Stats снимок метрик оркестратора.
type Stats struct {
	EventCounts   map[string]int64        `json:"event_counts"`
	ErrorCount    int64                   `json:"error_count"`
	LatencyAvgMs  map[string]float64      `json:"latency_avg_ms"`
	ModuleHealth  map[string]time.Time    `json:"module_health"` // последний успешный heartbeat
	ChatHistories map[int64][]HistoryItem `json:"-"`             // не сериализуем целиком в проде
}

// Monitor собирает счётчики, задержки, здоровье модулей и историю по chat_id.
type Monitor struct {
	mu           sync.RWMutex
	eventCounts  map[string]int64
	errorCount   int64
	latencySum   map[string]int64
	latencyN     map[string]int64
	moduleHealth map[string]time.Time
	chatHistory  map[int64][]HistoryItem
}

// NewMonitor создаёт монитор.
func NewMonitor() *Monitor {
	return &Monitor{
		eventCounts:  make(map[string]int64),
		latencySum:   make(map[string]int64),
		latencyN:     make(map[string]int64),
		moduleHealth: make(map[string]time.Time),
		chatHistory:  make(map[int64][]HistoryItem),
	}
}

// RecordEvent увеличивает счётчик по имени события.
func (m *Monitor) RecordEvent(name coreevents.Name) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCounts[string(name)]++
}

// RecordError фиксирует ошибку конвейера.
func (m *Monitor) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
}

// RecordLatency добавляет измерение задержки (мс) по типу шага.
func (m *Monitor) RecordLatency(step string, d time.Duration) {
	if step == "" {
		return
	}
	ms := d.Milliseconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencySum[step] += ms
	m.latencyN[step]++
}

// TouchModule отмечает успешную активность модуля.
func (m *Monitor) TouchModule(module string) {
	if module == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.moduleHealth[module] = time.Now()
}

// AppendChatHistory добавляет запись и обрезает историю до maxChatHistory.
func (m *Monitor) AppendChatHistory(chatID int64, item HistoryItem) {
	if chatID == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	h := append(m.chatHistory[chatID], item)
	if len(h) > maxChatHistory {
		h = h[len(h)-maxChatHistory:]
	}
	m.chatHistory[chatID] = h
}

// ChatHistory возвращает копию истории по чату.
func (m *Monitor) ChatHistory(chatID int64) []HistoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.chatHistory[chatID]
	out := make([]HistoryItem, len(src))
	copy(out, src)
	return out
}

// Snapshot возвращает агрегированные метрики.
func (m *Monitor) Snapshot() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	avg := make(map[string]float64)
	for k := range m.latencySum {
		n := m.latencyN[k]
		if n == 0 {
			continue
		}
		avg[k] = float64(m.latencySum[k]) / float64(n)
	}
	ec := make(map[string]int64, len(m.eventCounts))
	for k, v := range m.eventCounts {
		ec[k] = v
	}
	mh := make(map[string]time.Time, len(m.moduleHealth))
	for k, v := range m.moduleHealth {
		mh[k] = v
	}
	ch := make(map[int64][]HistoryItem)
	for k, v := range m.chatHistory {
		cp := make([]HistoryItem, len(v))
		copy(cp, v)
		ch[k] = cp
	}
	return Stats{
		EventCounts:   ec,
		ErrorCount:    m.errorCount,
		LatencyAvgMs:  avg,
		ModuleHealth:  mh,
		ChatHistories: ch,
	}
}
