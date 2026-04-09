package tracker

import "modulr/events"

// События модуля tracker.
const (
	EventPlanCreated      events.Name = "v1.tracker.plan.created"
	EventMilestoneReached events.Name = "v1.tracker.milestone.reached"
	EventHabitLogged      events.Name = "v1.tracker.habit.logged"
)
