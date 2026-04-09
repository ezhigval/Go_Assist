package ai

import "context"

// AIAPI публичный контракт AI-модуля.
type AIAPI interface {
	Start(ctx context.Context) error
	Stop() error
}
