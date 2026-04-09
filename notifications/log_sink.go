package notifications

import (
	"context"
	"log"
)

// LogSink заглушка доставки (dev).
type LogSink struct{}

// Send пишет уведомление в лог.
func (LogSink) Send(_ context.Context, n Notification) error {
	log.Printf("notify [%s] -> %s: %s | %s (trace=%s)", n.Channel, n.Target, n.Title, n.Body, n.TraceID)
	return nil
}
