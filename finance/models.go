package finance

import (
	"time"

	"modulr/events"
)

// TransactionType тип движения средств.
type TransactionType string

const (
	TransactionIncome   TransactionType = "income"
	TransactionExpense  TransactionType = "expense"
	TransactionTransfer TransactionType = "transfer"
)

// Transaction финансовая операция.
type Transaction struct {
	events.EntityBase
	Type            TransactionType `json:"type"`
	AmountMinor     int64           `json:"amount_minor"` // в минимальных единицах валюты
	Currency        string          `json:"currency"`
	Category        string          `json:"category"`
	Counterparty    string          `json:"counterparty"`
	Memo            string          `json:"memo"`
	LinkedEntityIDs []string        `json:"linked_entity_ids"`
}

// Subscription периодический платёж.
type Subscription struct {
	events.EntityBase
	Name         string    `json:"name"`
	AmountMinor  int64     `json:"amount_minor"`
	Currency     string    `json:"currency"`
	Cadence      string    `json:"cadence"` // monthly, yearly
	NextDue      time.Time `json:"next_due"`
	Active       bool      `json:"active"`
	AutoCategory string    `json:"auto_category"`
}

// Credit кредитная линия / долг.
type Credit struct {
	events.EntityBase
	Title         string    `json:"title"`
	LimitMinor    int64     `json:"limit_minor"`
	BalanceMinor  int64     `json:"balance_minor"`
	Currency      string    `json:"currency"`
	APRPercent    float64   `json:"apr_percent"`
	NextPaymentAt time.Time `json:"next_payment_at"`
}

// Investment позиция / вклад.
type Investment struct {
	events.EntityBase
	Symbol      string  `json:"symbol"`
	Units       float64 `json:"units"`
	CostBasis   float64 `json:"cost_basis"`
	MarketValue float64 `json:"market_value"`
	Currency    string  `json:"currency"`
}

// BudgetSnapshot снимок лимита по категории в сегменте.
type BudgetSnapshot struct {
	Segment  events.Segment `json:"segment"`
	Category string         `json:"category"`
	Limit    int64          `json:"limit_minor"`
	Spent    int64          `json:"spent_minor"`
}
