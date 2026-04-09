package notifications

import "context"

// Notification исходящее сообщение в канал (push/email/in-app — реализует Sink).
type Notification struct {
	Channel string
	Target  string
	Title   string
	Body    string
	TraceID string
}

// Sink доставка без блокировки шины (реализация — telegram, email, лог).
type Sink interface {
	Send(ctx context.Context, n Notification) error
}

// NotificationsAPI жизненный цикл подписок.
type NotificationsAPI interface {
	Start(ctx context.Context) error
	Stop() error
}
