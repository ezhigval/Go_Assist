UPDATE sessions
SET payload = jsonb_set(
    COALESCE(payload, '{}'::jsonb),
    '{_active_scope}',
    to_jsonb(active_scope::text),
    true
);

DROP INDEX IF EXISTS idx_sessions_active_scope;

ALTER TABLE sessions
    DROP CONSTRAINT IF EXISTS sessions_active_scope_check;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS active_scope;
