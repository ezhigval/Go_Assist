package scheduler

import (
	"context"
	"time"

	"modulr/events"
)

// SchedulerAPI публичный контракт планировщика.
type SchedulerAPI interface {
	ScheduleAt(ctx context.Context, fireAt time.Time, target events.Name, payload any) (jobID string, err error)
	Cancel(jobID string) bool
	Start(ctx context.Context) error
	Stop() error
}
