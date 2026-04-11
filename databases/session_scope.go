package databases

import "modulr/events"

const sessionActiveScopePayloadKey = "_active_scope"

func extractSessionActiveScope(payload map[string]interface{}) (string, map[string]interface{}) {
	scope := string(events.DefaultSegment())
	if len(payload) == 0 {
		return scope, nil
	}

	cleaned := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		if k == sessionActiveScopePayloadKey {
			if seg := events.ParseSegmentFromAny(v); seg != "" {
				scope = string(seg)
			}
			continue
		}
		cleaned[k] = v
	}
	if len(cleaned) == 0 {
		return scope, nil
	}
	return scope, cleaned
}

func hydrateSessionPayload(payload map[string]interface{}, activeScope string) map[string]interface{} {
	seg := events.ParseSegmentFromAny(activeScope)
	if seg == "" {
		seg = events.DefaultSegment()
	}
	if len(payload) == 0 {
		return map[string]interface{}{sessionActiveScopePayloadKey: string(seg)}
	}
	out := make(map[string]interface{}, len(payload)+1)
	for k, v := range payload {
		out[k] = v
	}
	out[sessionActiveScopePayloadKey] = string(seg)
	return out
}
