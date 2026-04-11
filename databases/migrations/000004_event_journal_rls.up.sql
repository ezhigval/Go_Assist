CREATE OR REPLACE FUNCTION modulr_scope_bypass_enabled() RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT COALESCE(NULLIF(current_setting('modulr.scope_bypass', true), ''), 'off') = 'on';
$$;

CREATE OR REPLACE FUNCTION modulr_allowed_scopes() RETURNS text[]
LANGUAGE sql
STABLE
AS $$
    SELECT COALESCE(
        array_remove(string_to_array(NULLIF(current_setting('modulr.allowed_scopes', true), ''), ','), ''),
        ARRAY[]::text[]
    );
$$;

CREATE OR REPLACE FUNCTION modulr_scope_allowed(target_scope text) RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT modulr_scope_bypass_enabled() OR COALESCE(target_scope = ANY(modulr_allowed_scopes()), false);
$$;

ALTER TABLE event_journal ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_journal FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS event_journal_scope_select ON event_journal;
DROP POLICY IF EXISTS event_journal_scope_insert ON event_journal;

CREATE POLICY event_journal_scope_select
    ON event_journal
    FOR SELECT
    USING (modulr_scope_allowed(scope));

CREATE POLICY event_journal_scope_insert
    ON event_journal
    FOR INSERT
    WITH CHECK (modulr_scope_allowed(scope));
