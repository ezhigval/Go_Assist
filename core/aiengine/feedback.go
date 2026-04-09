package aiengine

import (
	"fmt"
	"log"
	"sync"
)

// feedbackState внутреннее состояние весов моделей (только метаданные, без PII).
type feedbackState struct {
	mu      sync.RWMutex
	weights map[string]float64
	errors  map[string]int64
}

func newFeedbackState() *feedbackState {
	return &feedbackState{
		weights: make(map[string]float64),
		errors:  make(map[string]int64),
	}
}

// ApplyFeedback корректирует вес модели по результату исполнения решения.
func (s *feedbackState) ApplyFeedback(fb Feedback) {
	if fb.ModelID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delta := 0.05
	if !fb.OK {
		s.errors[fb.ModelID]++
		delta = -0.08
	}
	w := s.weights[fb.ModelID]
	if w == 0 {
		w = 1.0
	}
	w += delta
	if w < 0.1 {
		w = 0.1
	}
	if w > 2.0 {
		w = 2.0
	}
	s.weights[fb.ModelID] = w
	log.Printf("aiengine: feedback model=%s decision=%s ok=%v weight=%.3f", fb.ModelID, fb.DecisionID, fb.OK, w)
}

// Weight возвращает текущий вес модели.
func (s *feedbackState) Weight(modelID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if w, ok := s.weights[modelID]; ok && w > 0 {
		return w
	}
	return 1.0
}

// Snapshot копия весов для мониторинга.
func (s *feedbackState) Snapshot() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]float64, len(s.weights))
	for k, v := range s.weights {
		out[k] = v
	}
	return out
}

// ValidateFeedback проверяет поля фидбека.
func ValidateFeedback(fb Feedback) error {
	if fb.ModelID == "" {
		return fmt.Errorf("aiengine: empty model_id")
	}
	return nil
}
