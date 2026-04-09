package knowledge

import "modulr/events"

// События модуля knowledge.
const (
	EventSaved          events.Name = "v1.knowledge.saved"
	EventQuery          events.Name = "v1.knowledge.query"
	EventRecommendation events.Name = "v1.knowledge.recommendation"
)
