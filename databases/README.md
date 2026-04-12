# Databases Module

`databases/` отвечает за PostgreSQL-хранилище для transport/session сценариев и сервисной статистики Modulr.

## Текущее покрытие

- `users` — Telegram-пользователи.
- `chats` — чаты и каналы.
- `sessions` — состояние transport-диалога по `chat_id`.
- `sessions.active_scope` — явный активный life scope transport-сессии; payload остаётся для временных данных и совместимости.
- `event_journal` — trace-связанный журнал transport/runtime событий для replay и отладки.
- `stats` — scope-aware action log для dev/staging и audit-подобных срезов.
- `auth_sessions` — DB-backed auth-сессии с hash-токенами, scope и `allowed_scopes`.

## Миграции

Канонический источник схемы теперь хранится в versioned SQL-файлах:

- `databases/migrations/*.up.sql`
- `databases/migrations/*.down.sql`
- служебная таблица `schema_migrations` хранит `version`, `name`, `checksum`, `applied_at`
- раннер использует `pg_advisory_lock`, чтобы два deploy-процесса не применяли миграции одновременно

Базовая transport/schema зафиксирована в `000001_initial_transport_schema`.

Дополнительные additive-миграции:

- `000002_event_journal_scope_indexes` — индексы под scope-aware replay журнала;
- `000003_sessions_active_scope` — явный `active_scope` в `sessions` с backfill из legacy payload `_active_scope`.
- `000004_event_journal_rls` — PostgreSQL RLS policy для `event_journal` с runtime context через `SET LOCAL modulr.allowed_scopes` / `modulr.scope_bypass`.
- `000005_stats_scope_rls` — `stats.scope` + PostgreSQL RLS policy для action log.
- `000006_sessions_auth_rls` — chat-bound policy для `sessions` и DB-backed `auth_sessions` с token-hash policy.

Основные команды:

```bash
go run ./cmd/databases up
go run ./cmd/databases status
go run ./cmd/databases rls-status
go run ./cmd/databases rls-status -require-effective
go run ./cmd/databases app-role-sql -role=modulr_app
go run ./cmd/databases stats -scope=personal
go run ./cmd/databases down -steps=1
go run ./cmd/databases journal -trace=<trace_id> -scope=personal
```

Прод-правило:

1. `DB_AUTO_MIGRATE=false` на staging/production.
2. Версионируемые SQL-миграции запускаются отдельным deployment step до старта приложения.
3. После успешного применения миграций только затем поднимаются `telegram/`, `auth/`, worker-процессы и другие потребители БД.

Локальный dev-flow:

1. Поднять PostgreSQL.
2. Применить миграции через `go run ./cmd/databases up`.
3. Для локального `serve`/startup можно оставить `DB_AUTO_MIGRATE=true`, чтобы `databases.InitDB` использовал тот же раннер.
4. Проверить состояние через `go run ./cmd/databases status`.

## Rollback

Номинальный rollback для additive-изменений:

1. Остановить новые writer-процессы (`telegram`, фоновые воркеры, batch-задачи).
2. Переключить трафик на предыдущую версию приложения.
3. Выполнить `go run ./cmd/databases down -steps=1` (или другое число шагов, если это заранее проверено).
4. Поднять предыдущую версию приложения только если она совместима с оставшейся схемой/данными.

Если миграция необратима, разрушает данные или требует manual data-fix, down-step должен быть описан в релизной документации, а штатным rollback остаётся восстановление из snapshot/backup.

## Интеграция с Telegram

`telegram/cmd/telegram` может использовать PostgreSQL-backed store для `state.Store`:

- `TELEGRAM_STATE_STORE=memory` — дефолтный режим.
- `TELEGRAM_STATE_STORE=postgres` — состояние чатов хранится в `sessions`, active scope пишется в отдельное поле `sessions.active_scope`, а runtime пишет trace-связанные записи в `event_journal`.

Этот режим использует `GetOrCreateChat` как минимальный bootstrap для FK `sessions.chat_id -> chats.tg_id`. Обогащение metadata чата (`title`, `type`) остаётся задачей transport sync.

## Event Journal Contract

`event_journal` хранит последовательность событий по `trace_id`:

- вход транспорта (`v1.message.received`, `status=accepted`);
- outcome оркестратора (`v1.orchestrator.decision.outcome`, `status=completed|failed`);
- fallback (`v1.orchestrator.fallback.requested`, `status=fallback`);
- transport timeout (`v1.transport.response.timeout`, `status=timeout`) при ожидании sync-ответа.

Read-side policy:

- запись в `event_journal` остаётся append-only;
- для чтения replay/диагностики следует использовать scope-aware path `(*DB).ListJournalEventsByTraceScoped` / `(*DB).ListJournalEventsByChatScoped`;
- `NewJournalScopeFilter(baseScope, tags, metadata)` повторяет runtime policy (`allowed_scopes`, `allow_scope:<segment>`) и не даёт случайно читать cross-scope записи;
- `FullJournalScopeFilter()` оставлен только для admin/deploy tooling и совместимости со старым `DatabaseAPI`.

DB-enforced policy:

- `000004_event_journal_rls` включает PostgreSQL Row Level Security для `event_journal`;
- repository-слой теперь оборачивает append/read journal paths в транзакцию и выставляет `SET LOCAL modulr.allowed_scopes=<csv>` или `SET LOCAL modulr.scope_bypass=on`;
- Go-side `WHERE scope = ANY(...)` сохранён как дополнительная защита и для обратной совместимости при rollback миграции;
- для реального прод-эффекта приложение должно подключаться **не** под PostgreSQL superuser: суперпользователь обходит RLS даже при `FORCE ROW LEVEL SECURITY`.

Диагностика готовности:

- `go run ./cmd/databases rls-status` показывает, под каким DB role работает текущее соединение;
- `go run ./cmd/databases rls-status -require-effective` удобно для CI/deploy preflight и завершится с non-zero exit code, если текущая роль не даёт реального RLS enforcement;
- команда проверяет `rolsuper`, `rolbypassrls`, `relrowsecurity`, `relforcerowsecurity` и наличие policy для `event_journal`, `stats`, `sessions`, `auth_sessions`;
- `databases.Start()` теперь логирует тот же readiness snapshot и явно предупреждает, если приложение подключено под superuser или BYPASSRLS role, либо одна из scope-bound таблиц не защищена policy;
- при `DB_REQUIRE_RLS_EFFECTIVE=true` startup завершится ошибкой, если storage RLS неэффективен для текущего DB role.

Bootstrap non-superuser app role:

- `go run ./cmd/databases app-role-sql -role=modulr_app` печатает минимальный SQL bootstrap для application login role;
- helper специально **не** применяет SQL автоматически: роль и grants должен подтвердить владелец БД / DBA;
- после переключения `DB_USER`/`DB_PASS` включи `DB_REQUIRE_RLS_EFFECTIVE=true` и проверь эффективный режим через `go run ./cmd/databases rls-status -require-effective`.

Sessions/Auth storage authorization:

- `sessions` теперь читаются и удаляются только в контексте `modulr.chat_id`, а запись дополнительно проходит `active_scope` check через `modulr.allowed_scopes`;
- repository-слой выставляет эти параметры автоматически: `GetSession/ClearSession` используют `chat_id`, `SetSession` комбинирует `chat_id + active_scope`;
- `auth_sessions` хранят `token_hash` вместо plain token, а DB store `NewAuthSessionStore(db)` реализует `modulr/auth.SessionStore` без прямой зависимости `auth -> databases`;
- `auth_sessions` защищены policy по `modulr.auth_token_hash`; insert/update дополнительно требуют разрешённый `scope`.
- `auth/reference.go` добавляет opaque `SessionReference(token)`: transport может хранить только reference и валидировать/ревокать сессию без raw token.

Интеграция с Auth:

- для `auth.NewService(...)` можно передать `databases.NewAuthSessionStore(db)` как внешний `SessionStore`;
- `auth.Session` теперь scope-aware: базовый `scope` и `allowed_scopes` сохраняются в БД и прокидываются в runtime context через `auth.EnrichContext`.
- `telegram/cmd/telegram` уже использует этот path: `/login`, `/whoami`, `/logout` работают поверх `auth_sessions`, а state payload хранит только `_auth_session_ref`;
- `TELEGRAM_AUTH_REQUIRED=true` делает auth обязательной для обычных текстовых сообщений; при этом runtime получает `user_id/roles/allowed_scopes` из auth-сессии и отдельные `transport_user_id/transport_username`.

Stats scope contract:

- `LogAction` теперь читает reserved key `_scope` из metadata и пишет нормализованный scope в колонку `stats.scope`;
- если `_scope` не задан, используется `personal`;
- `go run ./cmd/databases stats -scope=personal [-allow-scopes=business]` возвращает агрегаты только по видимым scope;
- `000005_stats_scope_rls` повторяет тот же DB-enforced policy path, что и `event_journal`.

Минимальный запрос для replay одного trace:

```sql
SELECT trace_id, event_name, status, source, payload, metadata, created_at
FROM event_journal
WHERE trace_id = $1
ORDER BY created_at ASC, id ASC;
```

Под scope-aware replay на PostgreSQL добавлены индексы `000002_event_journal_scope_indexes`:

- `idx_event_journal_trace_scope_created_at`;
- `idx_event_journal_chat_scope_created_at`.

CLI-примеры:

```bash
# Показать один trace в рамках personal scope
go run ./cmd/databases journal -trace=trace-123 -scope=personal

# Разрешить контролируемый cross-scope replay
go run ./cmd/databases journal -trace=trace-123 -scope=personal -allow-scopes=business,travel

# Админский unrestricted read-path (только для deploy/tooling)
go run ./cmd/databases journal -chat=100500 -all-scopes -limit=20
```
