package aiengine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"modulr/events"
)

var _ AIEngine = (*Engine)(nil)

// Engine реализует AIEngine: маршрутизация моделей, таймауты, заглушка инференса.
type Engine struct {
	mu      sync.RWMutex
	router  *ModelRouter
	fb      *feedbackState
	cancel  context.CancelFunc
	started bool
}

// NewEngine создаёт движок с роутером по умолчанию.
func NewEngine() *Engine {
	return &Engine{
		router: NewModelRouter(),
		fb:     newFeedbackState(),
	}
}

// Start запускает фоновые задачи (резерв под health-check моделей).
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}
	cctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.started = true
	go e.runHealthLoop(cctx)
	log.Println("aiengine: started")
	return nil
}

func (e *Engine) runHealthLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// STUB: Model health monitoring requires ModelSpec baseURL/client fields for provider pings, status updates, and v1.ai.model.status events on degradation.
		}
	}
}

// Stop останавливает фоновые циклы.
func (e *Engine) Stop(ctx context.Context) error {
	_ = ctx
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
	e.started = false
	log.Println("aiengine: stopped")
	return nil
}

// RegisterModel регистрирует модель в роутере.
func (e *Engine) RegisterModel(ctx context.Context, spec ModelSpec) error {
	_ = ctx
	if spec.ID == "" {
		return fmt.Errorf("aiengine: empty model id")
	}
	e.router.Upsert(spec)
	return nil
}

// Analyze строит решения: выбор моделей → (заглушка) инференс → нормализация confidence с учётом весов.
func (e *Engine) Analyze(ctx context.Context, req Request) ([]Decision, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if req.Scope != "" && !events.IsValidSegment(events.Segment(req.Scope)) {
		return nil, fmt.Errorf("aiengine: unsupported scope %q", req.Scope)
	}
	models := e.router.Select(req)
	if len(models) == 0 {
		return nil, fmt.Errorf("aiengine: no models available")
	}
	// STUB: Real inference requires provider calls per ModelSpec with context timeout/cancellation, aggregating []Decision without raw text storage.
	decs := e.stubInfer(ctx, req, models)
	for i := range decs {
		mid, _ := decs[i].Parameters["_model_id"].(string)
		w := e.fb.Weight(mid)
		decs[i].Confidence = clamp01(decs[i].Confidence * w)
		delete(decs[i].Parameters, "_model_id")
	}
	return decs, nil
}

func clamp01(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < 0 {
		return 0
	}
	return x
}

// stubInfer детерминированная заглушка вместо реального инференса.
func (e *Engine) stubInfer(ctx context.Context, req Request, models []ModelSpec) []Decision {
	_ = ctx
	_ = e
	now := time.Now()
	var out []Decision
	primary := "llm_primary"
	if len(models) > 0 {
		primary = models[0].ID
	}
	text := strings.ToLower(req.Text)
	if strings.Contains(text, "встреч") || strings.Contains(text, "календар") || strings.Contains(text, "созвон") || req.KindHint == "calendar" {
		out = append(out, Decision{
			ID:         newDecisionID(),
			Target:     "calendar",
			Action:     "create_event",
			Parameters: map[string]any{"title": trimSnippet(req.Text, 80), "_model_id": primary},
			Confidence: 0.95,
			Scope:      effectiveScope(req),
			CreatedAt:  now,
		})
	}
	if strings.Contains(text, "напомин") || strings.Contains(text, "deadline") || req.KindHint == "reminder" {
		out = append(out, Decision{
			ID:         newDecisionID(),
			Target:     "tracker",
			Action:     "create_reminder",
			Parameters: map[string]any{"note": trimSnippet(req.Text, 120), "_model_id": primary},
			Confidence: 0.88,
			Scope:      effectiveScope(req),
			CreatedAt:  now,
		})
	}
	if strings.Contains(text, "маршрут") || strings.Contains(text, "route") || strings.Contains(text, "ехать") || req.KindHint == "route" {
		out = append(out, Decision{
			ID:         newDecisionID(),
			Target:     "maps",
			Action:     "build_route",
			Parameters: map[string]any{"query": trimSnippet(req.Text, 120), "_model_id": "route_planner"},
			Confidence: 0.91,
			Scope:      effectiveScope(req),
			CreatedAt:  now,
		})
	}
	if len(out) == 0 {
		out = append(out, Decision{
			ID:         newDecisionID(),
			Target:     "knowledge",
			Action:     "save_query",
			Parameters: map[string]any{"text": trimSnippet(req.Text, 200), "_model_id": primary},
			Confidence: 0.55,
			Scope:      effectiveScope(req),
			CreatedAt:  now,
		})
	}
	return out
}

func effectiveScope(req Request) string {
	if req.Scope != "" {
		return req.Scope
	}
	return "personal"
}

func trimSnippet(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func newDecisionID() string {
	return fmt.Sprintf("dec_%d", time.Now().UnixNano())
}

// Feedback применяет обратную связь к весам.
func (e *Engine) Feedback(ctx context.Context, fb Feedback) error {
	_ = ctx
	if err := ValidateFeedback(fb); err != nil {
		return err
	}
	e.fb.ApplyFeedback(fb)
	return nil
}
