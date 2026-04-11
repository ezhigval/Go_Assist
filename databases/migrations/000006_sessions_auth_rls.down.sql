DROP POLICY IF EXISTS auth_sessions_token_write ON auth_sessions;
DROP POLICY IF EXISTS auth_sessions_token_select ON auth_sessions;

ALTER TABLE IF EXISTS auth_sessions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS auth_sessions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sessions_chat_write ON sessions;
DROP POLICY IF EXISTS sessions_chat_select ON sessions;

ALTER TABLE IF EXISTS sessions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS sessions DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS auth_sessions;

DROP FUNCTION IF EXISTS modulr_auth_token_allowed(text);
DROP FUNCTION IF EXISTS modulr_request_auth_token_hash();
DROP FUNCTION IF EXISTS modulr_chat_allowed(bigint);
DROP FUNCTION IF EXISTS modulr_request_chat_id();
