CREATE INDEX IF NOT EXISTS idx_event_journal_trace_scope_created_at
    ON event_journal(trace_id, scope, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_event_journal_chat_scope_created_at
    ON event_journal(chat_id, scope, created_at DESC, id DESC);
