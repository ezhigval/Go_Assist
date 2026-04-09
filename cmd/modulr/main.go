// Сборочный щит: подключаешь модули в порядке внедрения, ядро шины одно.
package main

import (
	"context"
	"log"
	"time"

	"modulr/ai"
	"modulr/auth"
	"modulr/events"
	"modulr/files"
	"modulr/metrics"
	"modulr/notifications"
	"modulr/scheduler"
)

func main() {
	ctx := context.Background()
	bus := events.NewBus(128)

	idemSched := events.NewMemoryIdempotency()
	idemFiles := events.NewMemoryIdempotency()
	idemMetrics := events.NewMemoryIdempotency()

	authAPI := auth.NewService(auth.LoadConfig(), auth.NewMemorySessionStore(), bus)
	schedAPI := scheduler.NewService(scheduler.LoadConfig(), bus, idemSched)
	filesAPI := files.NewService(files.LoadConfig(), bus, idemFiles)
	notifyAPI := notifications.NewService(notifications.LoadConfig(), bus, notifications.LogSink{})
	metricsCfg := metrics.LoadConfig()
	metricsAPI := metrics.NewService(metricsCfg, bus, idemMetrics)
	aiAPI := ai.NewService(ai.LoadConfig(), bus, ai.StubGateway{})

	// Порядок: auth → scheduler → files → notifications → metrics → ai
	for _, m := range []interface{ Start(context.Context) error }{
		authAPI, schedAPI, filesAPI, notifyAPI, metricsAPI, aiAPI,
	} {
		if err := m.Start(ctx); err != nil {
			log.Fatalf("start module: %v", err)
		}
	}

	traceCtx := events.WithTraceID(ctx, "demo-trace-1")
	token, err := authAPI.CreateSession(traceCtx, "user_demo", []auth.Role{auth.RoleUser})
	if err != nil {
		log.Fatalf("session: %v", err)
	}
	sess, err := authAPI.ValidateToken(ctx, token)
	if err != nil {
		log.Fatalf("validate: %v", err)
	}
	log.Printf("auth: session ok user=%s", sess.UserID)

	bus.Publish(events.Event{
		Name:    events.V1SystemStartup,
		Payload: map[string]any{"component": "modulr"},
		Source:  "cmd/modulr",
		TraceID: events.TraceIDFromContext(traceCtx),
	})

	start := time.Now().Add(2 * time.Hour)
	bus.Publish(events.Event{
		ID:      "cal-1",
		Name:    events.V1CalendarCreated,
		Payload: scheduler.CalendarHint{ID: "cal-1", Title: "Синк", Start: start},
		Source:  "demo",
		TraceID: events.TraceIDFromContext(traceCtx),
		Context: map[string]any{"chat_id": int64(42), "user_id": sess.UserID},
	})

	bus.Publish(events.Event{
		ID:      "doc-1",
		Name:    events.V1TransportFileRecv,
		Payload: files.TransportFilePayload{FileName: "note.txt", MIME: "text/plain", Data: []byte("hello modulr"), ChatID: 42, UserID: sess.UserID},
		Source:  "demo",
		TraceID: events.TraceIDFromContext(traceCtx),
	})

	bus.Publish(events.Event{
		Name:    events.V1TodoDue,
		Payload: map[string]any{"todo_id": "t1"},
		Source:  "demo",
		TraceID: events.TraceIDFromContext(traceCtx),
		Context: map[string]any{"user_id": sess.UserID},
	})

	time.Sleep(400 * time.Millisecond)

	log.Printf("metrics: %+v", metricsAPI.Counts())
	log.Printf("dead-letter count: %d", bus.DeadLetterCount())

	_ = aiAPI.Stop()
	_ = metricsAPI.Stop()
	_ = notifyAPI.Stop()
	_ = filesAPI.Stop()
	_ = schedAPI.Stop()
	_ = authAPI.Stop()
}
