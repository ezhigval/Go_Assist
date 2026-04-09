package tracker

import (
	"context"
	"time"

	"modulr/events"
)

// TrackerAPI публичный контракт трекера.
type TrackerAPI interface {
	CreatePlan(ctx context.Context, p *Plan) error
	ReachMilestone(ctx context.Context, planID, milestoneID string) error
	LogHabit(ctx context.Context, habitID string, at time.Time) error
	AddChecklistItem(ctx context.Context, item *CheckListItem) error
	ListPlans(ctx context.Context, segment string) ([]Plan, error)
	RegisterSubscriptions(bus *events.EventBus)
}
