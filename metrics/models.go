package metrics

import "time"

// TraceSummary краткий observability-срез по trace_id.
type TraceSummary struct {
	TraceID    string    `json:"trace_id"`
	Scope      string    `json:"scope"`
	EventCount int64     `json:"event_count"`
	LastEvent  string    `json:"last_event"`
	LastSource string    `json:"last_source"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Snapshot агрегированный снимок metrics-модуля.
type Snapshot struct {
	Counts      map[string]int64 `json:"counts"`
	ScopeCounts map[string]int64 `json:"scope_counts"`
	Traces      []TraceSummary   `json:"traces"`
}
