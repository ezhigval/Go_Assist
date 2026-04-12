# Changelog Modulr

Формат записи аудита и крупных правок:

`- [YYYY-MM-DD] | путь/к файлу | суть изменения | правило PROJECT_RULES.md (раздел) или пометка`

Мелкие правки можно группировать одной строкой с перечислением файлов.

---

## [2026-04-11] v0.2: bus bridge и базовый тестовый контур

- [2026-04-11] | `core/busbridge/bridge.go` | **Создан** двунаправленный адаптер между `modulr/events` и `modulr/core/events`; переносит `trace_id/chat_id/scope/tags`, поддерживает alias core→domain и защиту от циклов | §2 A2, §5 V2
- [2026-04-11] | `core/events/bus.go` | Добавлен `SubscribeAll` для пассивных слушателей ядровой шины и bus bridge без ломки `EventBus` | §2 A2, §3 E2
- [2026-04-11] | `events/bus_test.go`, `events/segment_test.go`, `core/busbridge/bridge_test.go`, `core/orchestrator/registry_test.go`, `core/orchestrator/pipeline_test.go` | **Добавлены** тесты для критичных пакетов v0.2: доменная шина, сегменты, bridge, registry/pipeline | §4 C1, §8
- [2026-04-11] | `ARCHITECTURE.md`, `ROADMAP.md` | Документация синхронизирована с фактическим состоянием v0.2; EXC-001 закрыт, bridge описан в архитектуре | §7
- [2026-04-11] | `scripts/modulr-check.sh` / корневой модуль | Локальная валидация: `env GOCACHE=/tmp/go-build-cache go test ./...` и `env GOCACHE=/tmp/go-build-cache ./scripts/modulr-check.sh` прошли успешно | §4 C1, §7
- [2026-04-11] | `app/runtime.go`, `app/actions.go`, `app/runtime_test.go` | **Создан** runtime v0.3: transport-shaped ingress, orchestrator, bus bridge, доменные action handlers и E2E тест | §2 A2, §5 V3
- [2026-04-11] | `core/aiengine/types.go`, `core/aiengine/engine.go`, `core/orchestrator/pipeline.go` | `model_id` теперь проходит через decision/dispatch/outcome, что делает feedback loop корректным | §5 V3
- [2026-04-11] | `cmd/modulr/main.go`, `core/main.go`, `README.md`, `ROADMAP.md`, `ARCHITECTURE.md` | Quick-start и документация переведены на новый runtime message flow; отдельно зафиксировано, что прямой handoff из `telegram/` ещё не завершён | §7
- [2026-04-11] | `telegram/go.mod`, `telegram/cmd/telegram/main.go`, `telegram/modulr_ingress.go`, `telegram/bot.go`, `telegram/handler/router.go` | `telegram/` подключён к root runtime через local `replace`, восстановлен transport response flow и вынесен `cmd/telegram` вместо mixed layout в корне подмодуля | §2 A4, §5 V1
- [2026-04-11] | `telegram/handler/router_test.go`, `telegram/modulr_ingress_test.go`, `scripts/modulr-check.sh`, `PROJECT_RULES.md`, `README.md`, `ROADMAP.md` | Добавлены тесты и gate для `telegram/`; direct handoff из transport-слоя отмечен как выполненный шаг v0.3 | §4 C1, §7
- [2026-04-11] | `databases/cmd/databases/main.go`, `databases/api.go`, `databases/repo.go`, `databases/config.go`, `databases/README.md` | `databases/` переведён на importable layout `cmd/databases`, исправлен контракт `pgx/v5` (`pgconn.CommandTag`), добавлены `DB_AUTO_MIGRATE` и README с миграциями/rollback | §2 A4, §3 E1, §5 V1
- [2026-04-11] | `telegram/state/database.go`, `telegram/state/database_test.go`, `telegram/cmd/telegram/persistence.go`, `telegram/cmd/telegram/main.go`, `telegram/config.go`, `telegram/go.mod` | Добавлен PostgreSQL-backed `state.Store` для transport session и переключение `TELEGRAM_STATE_STORE=memory|postgres`; `telegram` теперь использует `databases.DatabaseAPI` через локальный bridge | §2 A4, §3 E5, §5 V1
- [2026-04-11] | `scripts/modulr-check.sh`, `PROJECT_RULES.md`, `ROADMAP.md` | Gate обновлены под новый статус `databases/`: модуль проходит `go test ./...`, roadmap v0.3 синхронизирован с миграционным README и DB-backed session store | §4 C1, §7
- [2026-04-11] | `databases/models.go`, `databases/repo.go`, `databases/migrations.go`, `databases/README.md`, `app/journal.go`, `app/runtime.go`, `app/runtime_test.go`, `telegram/cmd/telegram/persistence.go` | Реализован trace-связанный `event_journal`; `telegram` в PostgreSQL-режиме пишет входящие сообщения, outcome/fallback и timeout в журнал, а runtime подключает sink через интерфейс без прямой зависимости от `databases/` | §2 A4, §3 E5, §5 V1
- [2026-04-11] | `databases/migrations.go`, `databases/migrations/*.sql`, `databases/cmd/databases/main.go`, `databases/migrations_test.go`, `databases/README.md`, `README.md`, `ROADMAP.md` | Bootstrap `schemaSQL` заменён на versioned SQL migrations runner: `schema_migrations`, checksum drift-check, advisory lock, CLI `up/down/status`, отдельный deployment step и обновлённая документация v0.3 | §3 E1, §4 C1, §7
- [2026-04-11] | `EVENT_MATRIX.md`, `app/contracts.go`, `app/contracts_test.go`, `app/actions.go`, `app/runtime.go`, `app/runtime_test.go`, `telegram/modulr_ingress.go`, `README.md`, `ROADMAP.md` | Для v1.0 добавлена каноническая матрица событий `v1.*`, закреплены правила backward compatibility, а action event names и human-readable mapping вынесены в единый source of truth между runtime и transport | §2 A2, §5 V2, §7
- [2026-04-11] | `app/journal.go`, `app/runtime.go`, `app/runtime_test.go`, `telegram/modulr_ingress_test.go`, `core/aiengine/config.go`, `core/aiengine/provider.go`, `core/aiengine/engine.go`, `core/aiengine/engine_provider_test.go`, `ai/local_router/main.py`, `ai/pii.go`, `README.md`, `ai/AI_ARCHITECTURE.md`, `ROADMAP.md` | v1.0 усилен: добавлены tests для completed/fallback/timeout critical paths, `core/aiengine` получил gated `AI_PROVIDER=local` path c PII redaction и stub fallback, а docs приведены к фактическому runtime behaviour | §4 C1, §5 V2, §7
- [2026-04-11] | `RELEASE_TEMPLATE.md`, `README.md`, `ROADMAP.md` | Для v1.0 добавлен release notes / rollback template: preflight checks, миграционный шаг, переключение `AI_PROVIDER`/`AI_ALLOW_STUB_FALLBACK`, откат через предыдущую версию образа и down-step/snapshot для БД | §3 E1, §5 V1, §7
- [2026-04-11] | `events/scope_policy.go`, `core/orchestrator/pipeline.go`, `core/orchestrator/pipeline_test.go`, `app/runtime.go`, `app/runtime_test.go`, `metrics/service.go`, `metrics/models.go`, `metrics/service_test.go`, `ARCHITECTURE.md`, `ROADMAP.md` | Начат v1.5: orchestrator теперь блокирует cross-scope dispatch по умолчанию и пропускает его только при явной policy (`allowed_scopes` / `allow_scope:<segment>`), а runtime metrics собирают агрегаты по `scope` и краткие trace-summary по `trace_id` | §1 I4, §5 V2, §7
- [2026-04-11] | `metrics/api.go`, `metrics/README.md`, `core/main.go`, `cmd/modulr/main.go`, `README.md`, `ROADMAP.md` | Для v1.5 оформлен экспорт observability: публичный контракт `metrics.Snapshot`, quick-start логирует `scope_counts` и trace-summary, а документация зафиксировала различие между in-memory metrics и полным `event_journal` | §5 V1, §7
- [2026-04-11] | `databases/journal_scope.go`, `databases/journal_scope_test.go`, `databases/repo.go`, `databases/migrations/000002_event_journal_scope_indexes.*`, `databases/README.md`, `ARCHITECTURE.md`, `ROADMAP.md` | Для v1.5 storage-layer получил scope-aware read path для `event_journal`: `JournalScopeFilter` повторяет runtime policy (`allowed_scopes`, `allow_scope:<segment>`), scoped replay не ломает `DatabaseAPI`, а PostgreSQL получил индексы под `trace_id/chat_id + scope` | §1 I4, §5 V1, §7
- [2026-04-11] | `databases/cmd/databases/main.go`, `databases/README.md`, `ROADMAP.md` | Следующим шагом v1.5 операторский `databases` CLI получил команду `journal`: replay по `trace_id`/`chat_id` теперь использует `JournalScopeFilter`, а unrestricted read-path разрешён только через явный флаг `-all-scopes` | §1 I4, §7
- [2026-04-11] | `telegram/modulr_ingress.go`, `telegram/modulr_ingress_test.go`, `telegram/state/scope.go`, `telegram/state/scope_test.go`, `telegram/state/database.go`, `telegram/state/database_test.go`, `telegram/handler/router.go`, `telegram/handler/router_test.go`, `README.md`, `ROADMAP.md` | Продолжение v1.5 на transport-слое: Telegram ingress теперь берёт active scope из session payload, поддерживает `/scope <segment>` для переключения контекста по чату, а router/database store сохраняют payload-only state, чтобы persistent metadata не терялась между сообщениями | §1 I4, §5 V2, §7
- [2026-04-11] | `databases/session_scope.go`, `databases/session_scope_test.go`, `databases/repo.go`, `databases/models.go`, `databases/migrations/000003_sessions_active_scope.*`, `databases/README.md`, `ROADMAP.md` | Для v1.5 session-level persistence усилен на storage-слое: `sessions` получили явный `active_scope` с backfill из legacy `_active_scope`, rollback возвращает значение в payload, а `DatabaseAPI` сохранил обратную совместимость через гидратацию reserved key для `telegram/state` | §1 I4, §3 E5, §5 V1, §7
- [2026-04-11] | `databases/rls.go`, `databases/rls_test.go`, `databases/repo.go`, `databases/migrations/000004_event_journal_rls.*`, `databases/README.md`, `ARCHITECTURE.md`, `ROADMAP.md` | Следующий срез v1.5 перевёл `event_journal` на DB-enforced scope guard: PostgreSQL RLS policy использует `SET LOCAL modulr.allowed_scopes/modulr.scope_bypass`, append/read paths идут через scoped transaction helper, а Go-side filter оставлен как дополнительная защита и rollback-compatible fallback | §1 I4, §3 E5, §7
- [2026-04-11] | `databases/rls_status.go`, `databases/rls_status_test.go`, `databases/db.go`, `databases/cmd/databases/main.go`, `databases/README.md`, `ROADMAP.md` | Для operational readiness v1.5 диагностика RLS расширена с journal-only до storage-wide snapshot: `databases.Start()` и CLI `rls-status` теперь показывают effective ли policy для текущей DB role сразу по `event_journal` и `stats`, и явно предупреждают о superuser/BYPASSRLS bypass или неполной table policy | §3 E3, §7
- [2026-04-11] | `databases/app_role.go`, `databases/app_role_test.go`, `databases/cmd/databases/main.go`, `databases/README.md`, `ROADMAP.md` | Для app-role readiness v1.5 добавлен helper `app-role-sql`: `databases` CLI теперь печатает проверяемый SQL bootstrap для non-superuser application role и привязывает следующий ops-step к `rls-status`, а не к ручным заметкам в README | §3 E5, §7
- [2026-04-11] | `databases/stats_scope.go`, `databases/stats_scope_test.go`, `databases/repo.go`, `databases/models.go`, `databases/migrations/000005_stats_scope_rls.*`, `databases/cmd/databases/main.go`, `databases/README.md`, `ARCHITECTURE.md`, `ROADMAP.md` | Для следующего storage slice v1.5 `stats` переведён в scope-aware action log: `LogAction` пишет нормализованный scope через reserved key `_scope`, PostgreSQL RLS ограничивает `SELECT/INSERT`, а CLI `stats` показывает агрегаты только по разрешённым scope | §1 I4, §3 E5, §7
- [2026-04-11] | `databases/rls.go`, `databases/repo.go`, `databases/migrations/000006_sessions_auth_rls.*`, `databases/auth_store.go`, `databases/rls_status.go`, `databases/app_role.go`, `databases/README.md`, `auth/models.go`, `auth/service.go`, `auth/session_scope.go` | Следующий крупный storage/auth этап v1.5: введён единый storage access context (`allowed_scopes`, `chat_id`, `auth_token_hash`, bypass), `sessions` переведены на chat-bound RLS с write-check по `active_scope`, а `auth` получил DB-backed `auth_sessions` store с hash-токенами и scope-aware session model | §1 I4, §3 E5, §5 V1, §7
- [2026-04-12] | `auth/reference.go`, `auth/storage.go`, `auth/service.go`, `auth/memory_store.go`, `telegram/ingress_auth.go`, `telegram/modulr_ingress.go`, `telegram/state/auth.go`, `telegram/cmd/telegram/persistence.go`, `telegram/cmd/telegram/main.go`, `app/runtime.go`, `README.md`, `ROADMAP.md`, `ARCHITECTURE.md`, `databases/README.md` | DB-backed auth store подключён в реальный `telegram` transport: добавлены opaque session references, команды `/login` `/whoami` `/logout`, optional auth gate через `TELEGRAM_AUTH_REQUIRED`, а runtime metadata больше не затирает auth `user_id/roles/allowed_scopes` transport-полями | §1 I4, §3 E5, §5 V1, §7
- [2026-04-11] | `.env.example`, `config/config.example.yaml`, `README.md` | Root quick-start и env templates синхронизированы с фактическим `databases` config: вместо устаревшего `DB_DSN` задокументированы `DB_HOST/DB_PORT/DB_NAME/DB_USER/DB_PASS/DB_SSLMODE`, а app-role rollout теперь связан с `app-role-sql` и `rls-status` прямо в onboarding | §7
- [2026-04-11] | `frontend/package.json`, `frontend/package-lock.json` | Исправлен CI-конфликт peer dependencies: `react`/`react-dom` зафиксированы на `18.2.0`, а `@twa-dev/sdk` pinned до `7.0.0`, совместимых с `react-native@0.74.x`, чтобы `npm ci` в `frontend-check` не падал на `ERESOLVE` | §4 C1, §7

---

## [2026-04-10] Аудит: TODO, нормативка, автопроверка

- [2026-04-10] | `core/orchestrator/orchestrator.go` (~стр. 275) | TODO переименован в формат `TODO(modulr-orchestrator): … Action: …` | §3 E4
- [2026-04-10] | `core/aiengine/engine.go` (~56, 96) | TODO: health провайдеров и реальный инференс с явным Action; уточнение без поля Endpoint у ModelSpec | §3 E4, §2 A2
- [2026-04-10] | `core/aiengine/types.go` (~9) | Комментарий к `Request.Scope`: ссылка на полный список scope в `events/segment.go` | §1 I1, §5 V2
- [2026-04-10] | `finance/service.go` (~100, 132, 188, 307) | Четыре TODO приведены к формату modulr-finance + Action | §3 E4
- [2026-04-10] | `tracker/service.go` (~64, 109) | TODO modulr-tracker + Action (декомпозиция встречи, гео) | §3 E4
- [2026-04-10] | `knowledge/service.go` (~70, 155, 236) | TODO modulr-knowledge + Action | §3 E4
- [2026-04-10] | `email/service.go` (~147, 166) | TODO modulr-email + Action | §3 E4
- [2026-04-10] | `organizer/api.go` (~12) | TODO modulr-organizer (воркеры) + Action | §3 E4
- [2026-04-10] | `organizer/core.go` (~50, 121) | TODO динамические правила и ApplySuggestion + Action | §3 E4, §1 I2
- [2026-04-10] | `organizer/contacts/service.go` (~34) | TODO синхронизация контактов + Action (интерфейс SyncSource) | §3 E4, §5 V1
- [2026-04-10] | `organizer/todo/service.go` (~34) | TODO персистентность storage.Save + Action | §3 E4
- [2026-04-10] | `databases/repo.go` (~88) | TODO события аудита через EventPublisher, не Kafka в пакете | §3 E4, §1 I2
- [2026-04-10] | `databases/migrations.go` (~9) | TODO goose/migrate + Action | §3 E4
- [2026-04-10] | `telegram/examples/start_handler.go` (~23–24) | TODO метрики/меню + Action | §3 E4
- [2026-04-10] | `telegram/bot.go` (~132, 165) | TODO UX ошибки и EditMessage + Action (совместимость BotAPI сохранена) | §3 E4, §5 V1
- [2026-04-10] | `telegram/keyboard/templates.go` (~6) | TODO MenuProvider + Action | §3 E4
- [2026-04-10] | `PROJECT_RULES.md` | **Создан** полный свод правил и чеклист агента | §7
- [2026-04-10] | `CHANGELOG.md` | **Создан** журнал; эта запись | §7
- [2026-04-10] | `.cursor/rules/modulr.mdc` | **Создано** правило Cursor `alwaysApply` | §7

- [2026-04-10] | `organizer/` | Попытка вынести `main` в `cmd/organizer` **откатана**: цикл импортов `organizer → calendar → organizer`. Оставлен прежний `go run .`; в PROJECT_RULES и скрипте зафиксирован техдолг | §2 A4
- [2026-04-10] | `scripts/modulr-check.sh` | `go vet`+`build` только корень `modulr`; `gofmt` рекурсивно для `organizer/`, `telegram/`, `databases/` (везде смешанный layout с `main`) | §7, §3 C1
- [2026-04-10] | `README.md` | Ссылки на PROJECT_RULES, CHANGELOG, `modulr-check.sh`; примечание по `organizer` layout | §7

### Исключения из полного построчного чтения

- Каталоги `.idea/`, артефакты `data/files/**`, файлы `.env*` — не изменялись; секреты не аудировались содержимым.  
- При необходимости построчного аудита бинарных/IDE-файлов — отдельная задача.

---

## Прочие записи (история репозитория)

Предыдущие крупные изменения (scope LEGO, `metrics/`, `core/*`) зафиксированы в истории git/коммитов до введения CHANGELOG; при необходимости переноса старых событий — отдельный PR.
