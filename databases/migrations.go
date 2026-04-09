package databases

import (
	"context"
	"log"
)

// schemaSQL содержит SQL-скрипт для инициализации таблиц
// STUB: Schema versioning requires golang-migrate/goose as separate deployment step; keep runMigrations only for dev/bootstrap with explicit environment flag.
const schemaSQL = `
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

CREATE INDEX IF NOT EXISTS idx_users_tg_id ON users(tg_id);
CREATE INDEX IF NOT EXISTS idx_chats_tg_id ON chats(tg_id);
CREATE INDEX IF NOT EXISTS idx_sessions_chat_id ON sessions(chat_id);
CREATE INDEX IF NOT EXISTS idx_stats_user_id ON stats(user_id);
CREATE INDEX IF NOT EXISTS idx_stats_created_at ON stats(created_at);
`

// runMigrations применяет схему к БД
func (db *DB) runMigrations(ctx context.Context) error {
	log.Println("🗄️  Running database migrations...")
	if _, err := db.pool.Exec(ctx, schemaSQL); err != nil {
		return err
	}
	log.Println("✅ Migrations applied successfully")
	return nil
}
