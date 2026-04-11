DROP POLICY IF EXISTS stats_scope_insert ON stats;
DROP POLICY IF EXISTS stats_scope_select ON stats;

ALTER TABLE stats NO FORCE ROW LEVEL SECURITY;
ALTER TABLE stats DISABLE ROW LEVEL SECURITY;

UPDATE stats
SET metadata = jsonb_set(
    COALESCE(metadata, '{}'::jsonb),
    '{_scope}',
    to_jsonb(scope::text),
    true
);

DROP INDEX IF EXISTS idx_stats_scope_created_at;

ALTER TABLE stats
    DROP CONSTRAINT IF EXISTS stats_scope_check;

ALTER TABLE stats
    DROP COLUMN IF EXISTS scope;
