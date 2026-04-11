CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    tg_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    tg_id BIGINT UNIQUE NOT NULL,
    title VARCHAR(255),
    type VARCHAR(50) DEFAULT 'private',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT UNIQUE NOT NULL REFERENCES chats(tg_id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL DEFAULT 'idle',
    payload JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stats (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(tg_id),
    action VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_journal (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(128) NOT NULL,
    chat_id BIGINT NOT NULL,
    scope VARCHAR(64) NOT NULL DEFAULT 'personal',
    event_name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'accepted',
    source VARCHAR(64) NOT NULL,
    payload JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_tg_id ON users(tg_id);
CREATE INDEX IF NOT EXISTS idx_chats_tg_id ON chats(tg_id);
CREATE INDEX IF NOT EXISTS idx_sessions_chat_id ON sessions(chat_id);
CREATE INDEX IF NOT EXISTS idx_stats_user_id ON stats(user_id);
CREATE INDEX IF NOT EXISTS idx_stats_created_at ON stats(created_at);
CREATE INDEX IF NOT EXISTS idx_event_journal_trace_id ON event_journal(trace_id);
CREATE INDEX IF NOT EXISTS idx_event_journal_chat_id ON event_journal(chat_id);
CREATE INDEX IF NOT EXISTS idx_event_journal_created_at ON event_journal(created_at);
CREATE INDEX IF NOT EXISTS idx_event_journal_event_name ON event_journal(event_name);
