package state

import "modulr/events"

const activeScopePayloadKey = "_active_scope"

// ActiveScope возвращает активный scope из payload, если он валиден.
func ActiveScope(session Session) string {
	if len(session.Payload) == 0 {
		return ""
	}
	if scope := events.ParseSegmentFromAny(session.Payload[activeScopePayloadKey]); scope != "" {
		return string(scope)
	}
	return ""
}

// SetActiveScope сохраняет активный scope в payload сессии.
func SetActiveScope(session Session, scope string) Session {
	clone := Session{
		Key:     session.Key,
		Payload: clonePayload(session.Payload),
	}

	seg := events.ParseSegmentFromAny(scope)
	if seg == "" {
		if len(clone.Payload) == 0 {
			return clone
		}
		delete(clone.Payload, activeScopePayloadKey)
		if len(clone.Payload) == 0 {
			clone.Payload = nil
		}
		return clone
	}

	if clone.Payload == nil {
		clone.Payload = make(map[string]interface{})
	}
	clone.Payload[activeScopePayloadKey] = string(seg)
	return clone
}

// PreserveActiveScope переносит уже выбранный scope в следующий state, если
// обработчик не указал новый scope явно.
func PreserveActiveScope(current, next Session) Session {
	if ActiveScope(next) != "" {
		return next
	}

	scope := ActiveScope(current)
	if scope == "" {
		return next
	}
	return SetActiveScope(next, scope)
}
