package distributed

import coreevents "modulr/core/events"

// EnvelopeFromCoreEvent поднимает core/events.Event в transport-agnostic envelope.
func EnvelopeFromCoreEvent(evt coreevents.Event) Envelope {
	ctx := cloneMap(evt.Context)
	traceID, _ := ctx["trace_id"].(string)
	return Envelope{
		Name:    string(evt.Name),
		Payload: evt.Payload,
		Context: ctx,
		ChatID:  evt.ChatID,
		Scope:   evt.Scope,
		Tags:    cloneStrings(evt.Tags),
		TraceID: traceID,
	}
}

// CoreEvent возвращает обратно core/events.Event для текущего runtime.
func (e Envelope) CoreEvent() coreevents.Event {
	ctx := cloneMap(e.Context)
	if e.TraceID != "" {
		if ctx == nil {
			ctx = make(map[string]any, 1)
		}
		if _, exists := ctx["trace_id"]; !exists {
			ctx["trace_id"] = e.TraceID
		}
	}
	return coreevents.Event{
		Name:    coreevents.Name(e.Name),
		Payload: e.Payload,
		Context: ctx,
		ChatID:  e.ChatID,
		Scope:   e.Scope,
		Tags:    cloneStrings(e.Tags),
	}
}
