ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS active_scope VARCHAR(64) NOT NULL DEFAULT 'personal';

UPDATE sessions
SET active_scope = CASE LOWER(COALESCE(payload->>'_active_scope', ''))
    WHEN 'personal' THEN 'personal'
    WHEN 'family' THEN 'family'
    WHEN 'work' THEN 'work'
    WHEN 'business' THEN 'business'
    WHEN 'health' THEN 'health'
    WHEN 'travel' THEN 'travel'
    WHEN 'pets' THEN 'pets'
    WHEN 'assets' THEN 'assets'
    ELSE active_scope
END;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'sessions_active_scope_check'
    ) THEN
        ALTER TABLE sessions
            ADD CONSTRAINT sessions_active_scope_check
            CHECK (active_scope IN ('personal', 'family', 'work', 'business', 'health', 'travel', 'pets', 'assets'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_sessions_active_scope ON sessions(active_scope);
