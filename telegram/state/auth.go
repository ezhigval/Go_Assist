package state

import "strings"

const authSessionReferencePayloadKey = "_auth_session_ref"

// AuthSessionReference возвращает opaque reference auth-сессии из payload.
func AuthSessionReference(session Session) string {
	ref, _ := authSessionReferenceValue(session)
	return ref
}

// SetAuthSessionReference сохраняет opaque reference в payload.
// Пустая строка помечает явный logout/clear и не даёт router вернуть старое значение.
func SetAuthSessionReference(session Session, reference string) Session {
	clone := Session{
		Key:     session.Key,
		Payload: clonePayload(session.Payload),
	}
	if clone.Payload == nil {
		clone.Payload = make(map[string]interface{})
	}
	clone.Payload[authSessionReferencePayloadKey] = strings.TrimSpace(strings.ToLower(reference))
	return clone
}

// PreserveAuthSessionReference переносит auth reference в следующий state, если
// обработчик не указал новое значение явно.
func PreserveAuthSessionReference(current, next Session) Session {
	if _, ok := authSessionReferenceValue(next); ok {
		return next
	}
	ref := AuthSessionReference(current)
	if ref == "" {
		return next
	}
	return SetAuthSessionReference(next, ref)
}

func authSessionReferenceValue(session Session) (string, bool) {
	if len(session.Payload) == 0 {
		return "", false
	}
	raw, ok := session.Payload[authSessionReferencePayloadKey]
	if !ok {
		return "", false
	}
	value, _ := raw.(string)
	return strings.TrimSpace(strings.ToLower(value)), true
}
