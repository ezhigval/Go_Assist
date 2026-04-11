package databases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	err := db.pool.QueryRow(ctx, "SELECT id, chat_id, state, active_scope, payload, updated_at FROM sessions WHERE chat_id = $1", chatID).
		Scan(&s.ID, &s.ChatID, &s.State, &s.ActiveScope, &payloadBytes, &s.UpdatedAt)

	if err == pgx.ErrNoRows {
		// Возвращаем пустую сессию, а не ошибку
		s.ChatID = chatID
		s.State = "idle"
		s.ActiveScope = "personal"
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
	s.Payload = hydrateSessionPayload(s.Payload, s.ActiveScope)
	return s, nil
}

func (db *DB) SetSession(ctx context.Context, chatID int64, state string, payload map[string]interface{}) error {
	activeScope, cleanedPayload := extractSessionActiveScope(payload)
	payloadBytes, _ := json.Marshal(cleanedPayload)
	query := `
		INSERT INTO sessions (chat_id, state, active_scope, payload) VALUES ($1, $2, $3, $4)
		ON CONFLICT (chat_id) DO UPDATE SET state = $2, active_scope = $3, payload = $4, updated_at = NOW()`
	_, err := db.pool.Exec(ctx, query, chatID, state, activeScope, payloadBytes)
	return err
}

func (db *DB) ClearSession(ctx context.Context, chatID int64) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM sessions WHERE chat_id = $1", chatID)
	return err
}

// --- Журнал событий ---

func (db *DB) AppendJournalEvent(ctx context.Context, entry EventJournalEntry) (*EventJournalEntry, error) {
	if entry.TraceID == "" {
		return nil, fmt.Errorf("append journal event: trace_id is required")
	}
	if entry.EventName == "" {
		return nil, fmt.Errorf("append journal event: event_name is required")
	}
	if entry.Source == "" {
		return nil, fmt.Errorf("append journal event: source is required")
	}
	if entry.Scope == "" {
		entry.Scope = "personal"
	}
	if entry.Status == "" {
		entry.Status = "accepted"
	}

	payloadBytes, err := marshalJSONMap(entry.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal journal payload: %w", err)
	}
	metadataBytes, err := marshalJSONMap(entry.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal journal metadata: %w", err)
	}

	query := `
		INSERT INTO event_journal (trace_id, chat_id, scope, event_name, status, source, payload, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, trace_id, chat_id, scope, event_name, status, source, payload, metadata, created_at`

	stored := EventJournalEntry{}
	var storedPayload []byte
	var storedMetadata []byte
	access, err := singleScopeAccess(entry.Scope)
	if err != nil {
		return nil, err
	}

	err = db.withScopeAccess(ctx, access, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			entry.TraceID,
			entry.ChatID,
			entry.Scope,
			entry.EventName,
			entry.Status,
			entry.Source,
			payloadBytes,
			metadataBytes,
		).Scan(
			&stored.ID,
			&stored.TraceID,
			&stored.ChatID,
			&stored.Scope,
			&stored.EventName,
			&stored.Status,
			&stored.Source,
			&storedPayload,
			&storedMetadata,
			&stored.CreatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	stored.Payload, err = unmarshalJSONMap(storedPayload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal stored journal payload: %w", err)
	}
	stored.Metadata, err = unmarshalJSONMap(storedMetadata)
	if err != nil {
		return nil, fmt.Errorf("unmarshal stored journal metadata: %w", err)
	}
	return &stored, nil
}

func (db *DB) ListJournalEventsByTrace(ctx context.Context, traceID string, limit int) ([]EventJournalEntry, error) {
	return db.ListJournalEventsByTraceScoped(ctx, traceID, FullJournalScopeFilter(), limit)
}

// ListJournalEventsByTraceScoped возвращает trace replay только в пределах разрешённых scope.
func (db *DB) ListJournalEventsByTraceScoped(ctx context.Context, traceID string, filter JournalScopeFilter, limit int) ([]EventJournalEntry, error) {
	if traceID == "" {
		return nil, fmt.Errorf("list journal by trace: trace_id is required")
	}
	access, err := journalScopeAccess(filter)
	if err != nil {
		return nil, err
	}
	limit = normalizeJournalLimit(limit)

	var entries []EventJournalEntry
	if filter.Unrestricted() {
		err := db.withScopeAccess(ctx, access, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, `
				SELECT id, trace_id, chat_id, scope, event_name, status, source, payload, metadata, created_at
				FROM event_journal
				WHERE trace_id = $1
				ORDER BY created_at ASC, id ASC
				LIMIT $2
			`, traceID, limit)
			if err != nil {
				return err
			}
			defer rows.Close()

			entries, err = scanJournalRows(rows)
			return err
		})
		return entries, err
	}

	err = db.withScopeAccess(ctx, access, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, trace_id, chat_id, scope, event_name, status, source, payload, metadata, created_at
			FROM event_journal
			WHERE trace_id = $1
			  AND scope = ANY($2)
			ORDER BY created_at ASC, id ASC
			LIMIT $3
		`, traceID, filter.Scopes(), limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		entries, err = scanJournalRows(rows)
		return err
	})
	return entries, err
}

func (db *DB) ListJournalEventsByChat(ctx context.Context, chatID int64, limit int) ([]EventJournalEntry, error) {
	return db.ListJournalEventsByChatScoped(ctx, chatID, FullJournalScopeFilter(), limit)
}

// ListJournalEventsByChatScoped возвращает журнал чата только в пределах разрешённых scope.
func (db *DB) ListJournalEventsByChatScoped(ctx context.Context, chatID int64, filter JournalScopeFilter, limit int) ([]EventJournalEntry, error) {
	access, err := journalScopeAccess(filter)
	if err != nil {
		return nil, err
	}
	limit = normalizeJournalLimit(limit)

	var entries []EventJournalEntry
	if filter.Unrestricted() {
		err := db.withScopeAccess(ctx, access, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, `
				SELECT id, trace_id, chat_id, scope, event_name, status, source, payload, metadata, created_at
				FROM event_journal
				WHERE chat_id = $1
				ORDER BY created_at DESC, id DESC
				LIMIT $2
			`, chatID, limit)
			if err != nil {
				return err
			}
			defer rows.Close()

			entries, err = scanJournalRows(rows)
			return err
		})
		return entries, err
	}

	err = db.withScopeAccess(ctx, access, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, trace_id, chat_id, scope, event_name, status, source, payload, metadata, created_at
			FROM event_journal
			WHERE chat_id = $1
			  AND scope = ANY($2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`, chatID, filter.Scopes(), limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		entries, err = scanJournalRows(rows)
		return err
	})
	return entries, err
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

func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, query, args...)
}

func scanJournalRows(rows pgx.Rows) ([]EventJournalEntry, error) {
	var entries []EventJournalEntry
	for rows.Next() {
		entry, err := scanJournalEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func scanJournalEntry(rows pgx.Row) (EventJournalEntry, error) {
	entry := EventJournalEntry{}
	var payloadBytes []byte
	var metadataBytes []byte
	err := rows.Scan(
		&entry.ID,
		&entry.TraceID,
		&entry.ChatID,
		&entry.Scope,
		&entry.EventName,
		&entry.Status,
		&entry.Source,
		&payloadBytes,
		&metadataBytes,
		&entry.CreatedAt,
	)
	if err != nil {
		return EventJournalEntry{}, err
	}

	entry.Payload, err = unmarshalJSONMap(payloadBytes)
	if err != nil {
		return EventJournalEntry{}, fmt.Errorf("unmarshal journal payload: %w", err)
	}
	entry.Metadata, err = unmarshalJSONMap(metadataBytes)
	if err != nil {
		return EventJournalEntry{}, fmt.Errorf("unmarshal journal metadata: %w", err)
	}
	return entry, nil
}

func marshalJSONMap(m map[string]interface{}) ([]byte, error) {
	if len(m) == 0 {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

func unmarshalJSONMap(raw []byte) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]interface{})
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeJournalLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}
