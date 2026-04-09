package finance

import "modulr/events"

// Имена событий модуля finance (v1.finance.*).
const (
	EventTransactionCreated events.Name = "v1.finance.transaction.created"
	EventSubscriptionDue    events.Name = "v1.finance.subscription.due"
	EventBudgetExceeded     events.Name = "v1.finance.budget.exceeded"
)
