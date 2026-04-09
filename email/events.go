package email

import "modulr/events"

// События модуля email.
const (
	EventReceived       events.Name = "v1.email.received"
	EventActionRequired events.Name = "v1.email.action_required"
	EventSent           events.Name = "v1.email.sent"
)
