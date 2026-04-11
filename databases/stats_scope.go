package databases

import "modulr/events"

const statsScopeMetadataKey = "_scope"

func extractStatsScope(metadata map[string]interface{}) (string, map[string]interface{}) {
	scope := string(events.DefaultSegment())
	if len(metadata) == 0 {
		return scope, nil
	}

	if seg := events.ParseSegmentFromAny(metadata[statsScopeMetadataKey]); seg != "" {
		scope = string(seg)
	} else if seg := events.ParseSegmentFromAny(metadata["scope"]); seg != "" {
		scope = string(seg)
	}

	cleaned := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		if k == statsScopeMetadataKey {
			continue
		}
		cleaned[k] = v
	}
	if len(cleaned) == 0 {
		return scope, nil
	}
	return scope, cleaned
}
