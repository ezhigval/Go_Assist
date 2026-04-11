ALTER TABLE stats
    ADD COLUMN IF NOT EXISTS scope VARCHAR(64) NOT NULL DEFAULT 'personal';

UPDATE stats
SET scope = CASE LOWER(COALESCE(payload_scope.scope_value, ''))
    WHEN 'personal' THEN 'personal'
    WHEN 'family' THEN 'family'
    WHEN 'work' THEN 'work'
    WHEN 'business' THEN 'business'
    WHEN 'health' THEN 'health'
    WHEN 'travel' THEN 'travel'
    WHEN 'pets' THEN 'pets'
    WHEN 'assets' THEN 'assets'
    ELSE scope
END
FROM (
    SELECT id, COALESCE(metadata->>'_scope', metadata->>'scope') AS scope_value
    FROM stats
) AS payload_scope
WHERE payload_scope.id = stats.id;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'stats_scope_check'
    ) THEN
        ALTER TABLE stats
            ADD CONSTRAINT stats_scope_check
            CHECK (scope IN ('personal', 'family', 'work', 'business', 'health', 'travel', 'pets', 'assets'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_stats_scope_created_at ON stats(scope, created_at DESC, id DESC);

ALTER TABLE stats ENABLE ROW LEVEL SECURITY;
ALTER TABLE stats FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stats_scope_select ON stats;
DROP POLICY IF EXISTS stats_scope_insert ON stats;

CREATE POLICY stats_scope_select
    ON stats
    FOR SELECT
    USING (modulr_scope_allowed(scope));

CREATE POLICY stats_scope_insert
    ON stats
    FOR INSERT
    WITH CHECK (modulr_scope_allowed(scope));
