package email

import (
	"context"

	"modulr/events"
)

// EmailAPI публичный контракт email-модуля.
type EmailAPI interface {
	IngestIncoming(ctx context.Context, msg *EmailMessage) error
	SendOutgoing(ctx context.Context, msg *EmailMessage) error
	UpsertRule(ctx context.Context, r *Rule) error
	ListRules(ctx context.Context) ([]Rule, error)
	RegisterSubscriptions(bus *events.EventBus)
}
