package finance

import (
	"context"

	"modulr/events"
)

// FinanceAPI публичный контракт модуля (внешний мир вызывает только его; кросс-связи — через шину).
type FinanceAPI interface {
	CreateTransaction(ctx context.Context, t *Transaction) error
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	ListTransactions(ctx context.Context, segment string) ([]Transaction, error)
	UpsertSubscription(ctx context.Context, s *Subscription) error
	ListSubscriptions(ctx context.Context) ([]Subscription, error)
	UpsertCredit(ctx context.Context, c *Credit) error
	UpsertInvestment(ctx context.Context, inv *Investment) error
	BalanceBySegment(ctx context.Context, segment string) (int64, error)
	SetBudgetLimit(ctx context.Context, segment, category string, limitMinor int64) error
	RegisterSubscriptions(bus *events.EventBus)
}
