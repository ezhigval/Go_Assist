CREATE OR REPLACE FUNCTION modulr_request_chat_id() RETURNS bigint
LANGUAGE SQL
STABLE
AS $$
    SELECT NULLIF(current_setting('modulr.chat_id', true), '')::bigint;
$$;

CREATE OR REPLACE FUNCTION modulr_chat_allowed(target_chat_id bigint) RETURNS boolean
LANGUAGE SQL
STABLE
AS $$
    SELECT modulr_scope_bypass_enabled() OR COALESCE(target_chat_id = modulr_request_chat_id(), false);
$$;

CREATE OR REPLACE FUNCTION modulr_request_auth_token_hash() RETURNS text
LANGUAGE SQL
STABLE
AS $$
    SELECT NULLIF(current_setting('modulr.auth_token_hash', true), '');
$$;

CREATE OR REPLACE FUNCTION modulr_auth_token_allowed(target_token_hash text) RETURNS boolean
LANGUAGE SQL
STABLE
AS $$
    SELECT modulr_scope_bypass_enabled() OR COALESCE(target_token_hash = modulr_request_auth_token_hash(), false);
$$;

CREATE TABLE IF NOT EXISTS auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(128) NOT NULL,
    scope VARCHAR(64) NOT NULL DEFAULT 'personal',
    allowed_scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'auth_sessions_scope_check'
    ) THEN
        ALTER TABLE auth_sessions
            ADD CONSTRAINT auth_sessions_scope_check
            CHECK (scope IN ('personal', 'family', 'work', 'business', 'health', 'travel', 'pets', 'assets'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_scope_expires_at ON auth_sessions(scope, expires_at ASC, id ASC);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sessions_chat_select ON sessions;
DROP POLICY IF EXISTS sessions_chat_write ON sessions;

CREATE POLICY sessions_chat_select
    ON sessions
    FOR SELECT
    USING (modulr_chat_allowed(chat_id));

CREATE POLICY sessions_chat_write
    ON sessions
    FOR ALL
    USING (modulr_chat_allowed(chat_id))
    WITH CHECK (modulr_chat_allowed(chat_id) AND modulr_scope_allowed(active_scope));

ALTER TABLE auth_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_sessions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS auth_sessions_token_select ON auth_sessions;
DROP POLICY IF EXISTS auth_sessions_token_write ON auth_sessions;

CREATE POLICY auth_sessions_token_select
    ON auth_sessions
    FOR SELECT
    USING (modulr_auth_token_allowed(token_hash));

CREATE POLICY auth_sessions_token_write
    ON auth_sessions
    FOR ALL
    USING (modulr_auth_token_allowed(token_hash))
    WITH CHECK (modulr_auth_token_allowed(token_hash) AND modulr_scope_allowed(scope));
