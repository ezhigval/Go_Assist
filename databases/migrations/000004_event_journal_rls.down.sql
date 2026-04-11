DROP POLICY IF EXISTS event_journal_scope_insert ON event_journal;
DROP POLICY IF EXISTS event_journal_scope_select ON event_journal;

ALTER TABLE event_journal NO FORCE ROW LEVEL SECURITY;
ALTER TABLE event_journal DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS modulr_scope_allowed(text);
DROP FUNCTION IF EXISTS modulr_allowed_scopes();
DROP FUNCTION IF EXISTS modulr_scope_bypass_enabled();
