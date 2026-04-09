package databases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- Пользователи ---

func (db *DB) GetOrCreateUser(ctx context.Context, tgID int64, username string) (*User, error) {
	query := `
		INSERT INTO users (tg_id, username) VALUES ($1, $2)
		ON CONFLICT (tg_id) DO UPDATE SET username = $2, updated_at = NOW()
		RETURNING id, tg_id, username, created_at, updated_at`

	u := &User{}
	err := db.pool.QueryRow(ctx, query, tgID, username).Scan(&u.ID, &u.TgID, &u.Username, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (db *DB) UpdateUser(ctx context.Context, tgID int64, username string) error {
	_, err := db.pool.Exec(ctx, "UPDATE users SET username = $1, updated_at = NOW() WHERE tg_id = $2", username, tgID)
	return err
}

// --- Чаты ---

func (db *DB) GetOrCreateChat(ctx context.Context, tgID int64, title, chatType string) (*Chat, error) {
	query := `
		INSERT INTO chats (tg_id, title, type) VALUES ($1, $2, $3)
		ON CONFLICT (tg_id) DO UPDATE SET title = $2, type = $3
		RETURNING id, tg_id, title, type, created_at`

	c := &Chat{}
	err := db.pool.QueryRow(ctx, query, tgID, title, chatType).Scan(&c.ID, &c.TgID, &c.Title, &c.Type, &c.CreatedAt)
	return c, err
}

// --- Сессии ---

func (db *DB) GetSession(ctx context.Context, chatID int64) (*Session, error) {
	s := &Session{Payload: make(map[string]interface{})}
	var payloadBytes []byte

	err := db.pool.QueryRow(ctx, "SELECT id, chat_id, state, payload, updated_at FROM sessions WHERE chat_id = $1", chatID).
		Scan(&s.ID, &s.ChatID, &s.State, &payloadBytes, &s.UpdatedAt)

	if err == pgx.ErrNoRows {
		// Возвращаем пустую сессию, а не ошибку
		s.ChatID = chatID
		s.State = "idle"
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if len(payloadBytes) > 0 {
		if err := json.Unmarshal(payloadBytes, &s.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal session payload: %w", err)
		}
	}
	return s, nil
}

func (db *DB) SetSession(ctx context.Context, chatID int64, state string, payload map[string]interface{}) error {
	payloadBytes, _ := json.Marshal(payload)
	query := `
		INSERT INTO sessions (chat_id, state, payload) VALUES ($1, $2, $3)
		ON CONFLICT (chat_id) DO UPDATE SET state = $2, payload = $3, updated_at = NOW()`
	_, err := db.pool.Exec(ctx, query, chatID, state, payloadBytes)
	return err
}

func (db *DB) ClearSession(ctx context.Context, chatID int64) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM sessions WHERE chat_id = $1", chatID)
	return err
}

// --- Статистика ---

func (db *DB) LogAction(ctx context.Context, userID int64, action string, metadata map[string]interface{}) error {
	metaBytes, _ := json.Marshal(metadata)
	_, err := db.pool.Exec(ctx, "INSERT INTO stats (user_id, action, metadata) VALUES ($1, $2, $3)", userID, action, metaBytes)
	// STUB: Audit events require publishing anonymized events to modulr/events bus (v1.metrics.* / v1.audit.*) via EventPublisher interface, no direct Kafka dependency.
	return err
}

func (db *DB) GetStats(ctx context.Context) (*StatsSummary, error) {
	s := &StatsSummary{}
	query := `
		SELECT 
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM chats),
			(SELECT COUNT(*) FROM stats)`
	err := db.pool.QueryRow(ctx, query).Scan(&s.TotalUsers, &s.TotalChats, &s.TotalActions)
	return s, err
}

// --- Универсальные методы ---

func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, query, args...)
}

func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (pgx.CommandTag, error) {
	return db.pool.Exec(ctx, query, args...)
}
