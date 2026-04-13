# Modulr — продуктовая дорожная карта

| Поле | Значение |
|------|----------|
| **Владелец артефакта** | Lead Product Architect |
| **Последнее обновление** | 2026-04-12 |
| **Следующий чек-ап** | 2026-04-24 |
| **Модуль Go** | `modulr` (`go 1.21`) |

---

## Синхронизация с правилами проекта

| Источник правил | Статус |
|-----------------|--------|
| `PROJECT_RULES.md` | **В репозитории, синхронизирован с v0.2 на 2026-04-11** |

### Исключение с обоснованием

| ID | Отклонение | Обоснование | Компенсирующая мера |
|----|------------|-------------|---------------------|
| **EXC-001** | Закрыто 2026-04-10 | `PROJECT_RULES.md` создан и подключён к канону репозитория | Поддерживать синхронизацию через CHANGELOG и обновления ROADMAP/README/ARCHITECTURE |

Обязательные gate для кода: **PROJECT_RULES.md** + **ARCHITECTURE.md** + **README.md**.

---

## Версионирование событий и API

| Слой | Правило | Пример |
|------|---------|--------|
| **Имена событий** | Префикс мажорной версии доменной шины: `v1.{module}.{action}`; ядро — `modulr/core/events` (`v1.message.*`, `v1.orchestrator.*`, `v1.ai.*`) | `v1.finance.transaction.created` |
| **Breaking change** | Новая мажорная ветка имён (`v2.*`) или параллельная подписка + deprecation-цикл в ROADMAP | Таблица миграции в релиз-нотах |
| **Публичные Go API** | Минорные расширения без ломки; удаление/смена сигнатур — новая мажорная версия модуля или явный deprecation | Семвер на уровне релиза платформы |
| **Полезная нагрузка** | Обратная совместимость: новые поля опциональны; обязательные поля — только в новой версии события | JSON-схемы / примеры в `ARCHITECTURE.md` (по мере появления) |

---

## Зависимости между фазами (кратко)

```
v0.1 (фундамент) → v0.2 (правила + адаптер шин + тесты)
       → v0.3 (транспорт/БД в основном потоке) → v1.0 (MVP контракты)
       → v1.5 (изоляция scope, наблюдаемость) → v2.0 (распределение, UI, плагины)
```

Приоритеты не меняются без проверки: **шина и контракты** → **единый процесс (адаптер)** → **внешние каналы** → **прод-политики** → **масштабирование**.

---

## Фазы

### v0.1 — Фундамент платформы

| | |
|--|--|
| **Цель** | Рабочий каркас: доменная шина, scope-модель, ядро (оркестратор + aiengine), базовые модули и демо-сборка. |
| **Статус** | Основной объём выполнен (см. чек-лист). |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| `events/` | `Bus`, `Event`, `v1.*`, `Storage`, `MemoryStorage`, trace/idempotency |
| `events/segment.go` | Канонические `Segment` / scope |
| `core/orchestrator/` | Реестр, pipeline, API, monitor, fallback |
| `core/aiengine/` | Решения, фидбек, роутер, API (stub-интеграция помечена в коде) |
| `core/events/` | Шина ядра, константы имён |
| Доменные пакеты | `finance`, `tracker`, `knowledge`, `metrics`, `notifications`, `media`, `email`, `auth`, `scheduler`, `files`, `ai`, … |
| `organizer/`, `telegram/`, `databases/` | Отдельные `go.mod`, композит и транспорт/данные |
| `cmd/modulr/main.go` | Сборочный пример (часть модулей) |
| `README.md`, `ARCHITECTURE.md` | Продуктовый и технический канон |

**Валидация (Definition of Done)**

- [x] `go build ./...` на корневом модуле без ошибок
- [x] Демо: `go run ./cmd/modulr/` стартует и публикует события
- [x] Доменные модули не импортируют друг друга (сверка с ARCHITECTURE)
- [x] Имена событий согласованы с префиксом `v1.` на доменной шине

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Две шины расходятся по смыслу | v0.2: явный адаптер + таблица соответствия имён |
| Stub AI в проде | v1.0: feature-flag, отказ на `v1.orchestrator.fallback.requested` (уже заложено в архитектуре) |

---

### v0.2 — Правила, адаптер шин, качество

| | |
|--|--|
| **Цель** | Формализовать правила разработки, связать `core/events` ↔ `events` в одном процессе, поднять базовую планку тестов и регрессии. |
| **Статус** | Выполнено локально 2026-04-11 (без CI, с ручной валидацией) |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| `PROJECT_RULES.md` | Gate: проектирование, тесты, документация, миграции, откаты |
| `core/busbridge/` | Адаптер mirror/dispatch между шинами (как в ARCHITECTURE) |
| Тесты | `go test ./...` для критичных пакетов: `events`, orchestrator registry/pipeline, сегменты |
| Обновление `ARCHITECTURE.md` | Диаграмма потоков после адаптера |

**Реализовано 2026-04-11**

- `core/busbridge`: двунаправленный bridge между `modulr/events` и `modulr/core/events`;
- `core/events.MemoryBus.SubscribeAll`: пассивное наблюдение за всеми событиями без ломки `EventBus`;
- тесты: `events/bus_test.go`, `events/segment_test.go`, `core/busbridge/bridge_test.go`, `core/orchestrator/registry_test.go`, `core/orchestrator/pipeline_test.go`;
- валидация: `env GOCACHE=/tmp/go-build-cache go test ./...` и `env GOCACHE=/tmp/go-build-cache ./scripts/modulr-check.sh`.

**Валидация**

- [x] `PROJECT_RULES.md` принят и ссылка из README
- [x] Адаптер покрыт тестом «одно сообщение → обе шины видят согласованный trace»
- [x] CI или локальный скрипт: `go test ./...` (минимальный порог — без падений на защищённых пакетах)
- [x] ROADMAP: закрыты пункты v0.2, запись в Change Log

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Ограниченность alias-маршрутов | Расширять таблицу соответствий и payload-mapper'ы по мере подключения transport/E2E из v0.3 |
| Нет CI | Issue с владельцем и ETA; до CI — обязательный ручной `go test` перед merge |

**Трекинг**

| Задача | Версия | Примечание |
|--------|--------|------------|
| Создать `PROJECT_RULES.md` | v0.2 | Закрыто |
| Адаптер шин | v0.2 | Закрыто: `core/busbridge/` |
| Минимальный набор тестов | v0.2 | Закрыто |

---

### v0.3 — Транспорт и персистентность в основном сценарии

| | |
|--|--|
| **Цель** | Telegram (и при необходимости `databases`) предсказуемо подключены к оркестратору/шине в типовом сценарии «сообщение пользователя». |
| **Статус** | Выполнено локально 2026-04-11 (без CI, с ручной валидацией): единый runtime, direct handoff из `telegram/`, transport response flow, PostgreSQL-backed session store, trace-связанный event journal и versioned migrations runner реализованы. |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| `app/` + `cmd/*` | Единая точка входа с transport-shaped ingress + `core/orchestrator` + доменная шина (через адаптер v0.2) |
| `databases/` | Миграции и контракт хранения сессий/событий документированы |
| Документация | Поток из ARCHITECTURE проверен на реальной цепочке вызовов |

**Реализовано 2026-04-11**

- `app/runtime.go`: локальная сборка `core bus + domain bus + busbridge + orchestrator + scheduler + domain modules`;
- `app/actions.go`: исполнители для `v1.tracker.create_reminder`, `v1.tracker.create_task`, `v1.finance.create_transaction`, `v1.knowledge.save_query`, `v1.knowledge.save_note`;
- `core/aiengine/types.go`, `core/aiengine/engine.go`, `core/orchestrator/pipeline.go`: `model_id` теперь проходит через decision → dispatch → outcome, что замыкает feedback loop;
- `app/runtime_test.go`: E2E тест сценария `message -> orchestrator -> domain action -> decision.outcome`;
- `cmd/modulr/main.go` и `core/main.go`: quick-start теперь запускает v0.3 runtime и публикует demo-сообщение как вход транспорта.
- `telegram/go.mod`, `telegram/cmd/telegram/main.go`, `telegram/modulr_ingress.go`: подмодуль `telegram/` подключён к root runtime через local `replace modulr => ..` и default text handoff;
- `telegram/handler/router.go`, `telegram/bot.go`, `telegram/handler/router_test.go`, `telegram/modulr_ingress_test.go`: transport-layer снова возвращает и отправляет `Response`, подмодуль проходит `go test ./...`.
- `databases/cmd/databases/main.go`, `databases/README.md`, `databases/config.go`: `databases/` переведён на importable layout `cmd/databases`, добавлены `DB_AUTO_MIGRATE` и миграционный README с rollback-порядком;
- `telegram/state/database.go`, `telegram/cmd/telegram/persistence.go`: `telegram/` умеет переключать `state.Store` между `memory` и `postgres`, сохраняя transport session в `databases.DatabaseAPI`.
- `databases/models.go`, `databases/repo.go`, `databases/migrations.go`: добавлен `event_journal` с выборками по `trace_id` / `chat_id` для replay и диагностики основного transport flow;
- `app/journal.go`, `app/runtime.go`, `app/runtime_test.go`, `telegram/cmd/telegram/persistence.go`: runtime поддерживает опциональный journal sink, а `telegram` в PostgreSQL-режиме пишет `v1.message.received`, outcome/fallback и timeout в `event_journal`.
- `databases/migrations/*.sql`, `databases/migrations.go`, `databases/cmd/databases/main.go`, `databases/migrations_test.go`: bootstrap `schemaSQL` заменён на versioned migration runner с `schema_migrations`, checksum drift-check, advisory lock и CLI-командами `up/down/status`.

**Валидация**

- [x] E2E-сценарий: входящее сообщение → публикация на шине → (stub) решение → доменное исполнение → `v1.orchestrator.decision.outcome`
- [x] Прямой handoff из `telegram/` в ingress runtime без ручного demo-вызова
- [x] План миграций БД + откат описан в PROJECT_RULES / миграционном README пакета
- [x] Trace-связанный event journal сохраняет основной `telegram -> runtime -> outcome/fallback` путь
- [x] Нет обхода gate из PROJECT_RULES.md
- [x] Bootstrap-схема заменена на versioned migrations tool и вынесена в отдельный deployment step

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Разные `go.mod` | Для `telegram/` зафиксированы local `replace modulr => ..` и `replace databases => ../databases`; при масштабировании до >2 подмодулей допустим `go.work` |
| Сложность E2E | Сначала интеграционный тест с in-memory шиной |

---

### v1.0 — MVP продукта (стабильные контракты)

| | |
|--|--|
| **Цель** | Версионируемый «релиз платформы»: стабильные контракты событий для подключаемых модулей, понятная модель деплоя и конфигурации. |
| **Статус** | Выполнено локально 2026-04-11 (без production deploy): каноническая матрица `v1.*`, критические runtime/transport тесты, gated local AI integration и release rollback template реализованы. |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| Матрица событий `v1.*` | Таблица: модуль, событие, поля, producer/consumer |
| Реальная или гейтнутая AI-интеграция | Замена/обёртка над `// TODO: Real Model Integration` |
| Release notes шаблон | Breaking vs additive, миграции |
| Политика security | Секреты, логи, PII — согласовано с ARCHITECTURE |

**Реализовано 2026-04-11**

- `EVENT_MATRIX.md`: зафиксированы поддерживаемые contracts для `transport -> orchestrator -> runtime actions`, доменные `v1.*` события и правила backward compatibility;
- `app/contracts.go`, `telegram/modulr_ingress.go`: action event names и human-readable mapping вынесены в единый source of truth без дублирования строк в transport-слое.
- `app/runtime_test.go`, `telegram/modulr_ingress_test.go`: покрыты completed/fallback/timeout paths, а также action branches для `tracker`, `finance`, `knowledge`;
- `core/aiengine/config.go`, `core/aiengine/provider.go`, `core/aiengine/engine_provider_test.go`, `ai/local_router/main.py`: `core/aiengine` получил gated provider path `AI_PROVIDER=local` с PII redaction перед внешним вызовом и stub fallback при сбое/недоступности провайдера.
- `RELEASE_TEMPLATE.md`: зафиксированы release notes template, preflight checks и rollback order через feature flags (`AI_PROVIDER`, `AI_ALLOW_STUB_FALLBACK`) и версию образа/бинаря.

**Валидация**

- [x] Обратная совместимость v1 событий задокументирована
- [x] Критические пути покрыты тестами
- [x] Откат релиза описан (feature flags / версия образа)

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Скачок нагрузки | Очереди, лимиты на публикацию, dead-letter мониторинг |

---

### v1.5 — Изоляция scope и наблюдаемость

| | |
|--|--|
| **Цель** | Политики изоляции `scope` на уровне хранилища и авторизации; метрики/трейсы для прод-эксплуатации. |
| **Статус** | В работе. На 2026-04-12 закрыты runtime-level scope isolation, observability, scope-aware read-path для `event_journal`, session-level active scope persistence, DB-enforced RLS slices для `event_journal` + `stats`, storage authorization для `sessions` и DB-backed `auth_sessions`, реальное подключение auth store в `telegram` transport, runtime role-based admission control для AI decisions и fail-fast RLS preflight для окружений; следующим шагом остаётся фактический rollout non-superuser app-role по staging/production с проверкой deploy secrets/vars. |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| RLS / tenant / scope | Дизайн + реализация в `databases/` или слое репозиториев |
| Наблюдаемость | Связка `metrics` + трассировка trace_id по шине |
| Политики смешивания scope | Явные правила (README/ARCHITECTURE) реализованы в коде |

**Реализовано 2026-04-11/12**

- `events/scope_policy.go`, `core/orchestrator/pipeline.go`, `core/orchestrator/pipeline_test.go`: orchestrator по умолчанию режет cross-scope `Decision.Scope`; переход разрешается только при явной policy через `allowed_scopes` в metadata или tags вида `allow_scope:<segment>`;
- `app/runtime.go`, `app/runtime_test.go`: runtime подключает `metrics.Service`, а E2E-тесты покрывают запрет утечки между scope, разрешённый cross-scope path и trace-visible outcome;
- `metrics/service.go`, `metrics/models.go`, `metrics/service_test.go`: `metrics` теперь сохраняет не только общие счётчики событий, но и агрегаты по `scope` и краткие trace-summary по `trace_id`.
- `databases/journal_scope.go`, `databases/repo.go`, `databases/migrations/000002_event_journal_scope_indexes.*`, `databases/journal_scope_test.go`: storage-layer получил scope-aware read-path для `event_journal`; replay/диагностика теперь могут ограничиваться базовым `scope` и явными `allowed_scopes`, а PostgreSQL получил индексы под `trace_id/chat_id + scope`.
- `databases/cmd/databases/main.go`, `databases/README.md`: операторский CLI получил команду `journal`, которая использует тот же scope filter для безопасного replay по `trace_id`/`chat_id`; unrestricted доступ вынесен в явный флаг `-all-scopes`.
- `telegram/modulr_ingress.go`, `telegram/state/scope.go`, `telegram/state/database.go`, `telegram/handler/router.go`: Telegram transport перестал жёстко слать только `personal`; active scope теперь хранится в session payload, команда `/scope <segment>` переключает его по чату, а router сохраняет payload-only state для persistent metadata.
- `databases/session_scope.go`, `databases/repo.go`, `databases/migrations/000003_sessions_active_scope.*`, `databases/session_scope_test.go`, `databases/README.md`: active scope вынесен из legacy JSON payload в отдельное поле `sessions.active_scope` с backfill/rollback-совместимостью; `DatabaseAPI` при этом сохраняет прежний метод `SetSession`, а reserved payload key `_active_scope` гидратируется обратно для transport-слоя.
- `databases/rls.go`, `databases/repo.go`, `databases/migrations/000004_event_journal_rls.*`, `databases/rls_test.go`: `event_journal` переведён на первый DB-enforced RLS slice; repository выставляет `SET LOCAL modulr.allowed_scopes/modulr.scope_bypass` в транзакции, а PostgreSQL policy дублирует Go-side scope filter для replay и append paths.
- `databases/rls_status.go`, `databases/db.go`, `databases/cmd/databases/main.go`, `databases/rls_status_test.go`: для RLS добавлена readiness-диагностика; startup-лог и CLI `rls-status` теперь показывают, effective ли policy для текущего DB role сразу по `event_journal`, `stats`, `sessions`, `auth_sessions`, или соединение идёт под superuser/BYPASSRLS.
- `databases/app_role.go`, `databases/app_role_test.go`, `databases/cmd/databases/main.go`, `databases/README.md`: добавлен app-role bootstrap helper `app-role-sql`, чтобы переход на non-superuser DB role для реального RLS enforcement не зависел от ручных ad-hoc SQL заметок.
- `databases/stats_scope.go`, `databases/repo.go`, `databases/migrations/000005_stats_scope_rls.*`, `databases/models.go`, `databases/README.md`, `databases/cmd/databases/main.go`: `stats` получил scope column, DB-enforced RLS, reserved metadata key `_scope` и scoped CLI-агрегацию через `stats`, что расширило storage authorization за пределы `event_journal`.
- `databases/rls.go`, `databases/repo.go`, `databases/migrations/000006_sessions_auth_rls.*`, `databases/auth_store.go`, `databases/rls_status.go`, `databases/app_role.go`, `databases/README.md`, `auth/models.go`, `auth/service.go`, `auth/session_scope.go`: storage authorization расширен на `sessions` и новый `auth_sessions`; БД теперь понимает единый access context (`scope` + `chat_id` + `auth_token_hash`), `auth` получил scope-aware session model, а `databases.NewAuthSessionStore(db)` даёт DB-backed path без прямой зависимости `auth -> databases`.
- `auth/reference.go`, `auth/storage.go`, `auth/service.go`, `auth/memory_store.go`, `telegram/ingress_auth.go`, `telegram/modulr_ingress.go`, `telegram/state/auth.go`, `telegram/cmd/telegram/persistence.go`, `telegram/cmd/telegram/main.go`: DB-backed auth store подключён в реальный `telegram` entrypoint; transport хранит только opaque session reference, поддерживает `/login`, `/whoami`, `/logout`, может требовать auth через `TELEGRAM_AUTH_REQUIRED=true`, а runtime теперь сохраняет auth `user_id/roles/allowed_scopes` рядом с transport identity без затирания контекста.
- `events/access_policy.go`, `events/access_policy_test.go`, `auth/service.go`, `core/orchestrator/orchestrator.go`, `core/orchestrator/pipeline.go`, `core/orchestrator/pipeline_test.go`, `app/runtime_test.go`, `telegram/modulr_ingress.go`: runtime auth-policy вынесен в общий helper-слой; `telegram` ingress помечает `auth_required`, orchestrator отклоняет AI decisions без ролей или с запрещённой ролью, а fallback теперь несёт summary причин (`role_denied`, `auth_required`, и т.д.) для диагностики.
- `databases/config.go`, `databases/rls_enforcement.go`, `databases/db.go`, `databases/cmd/databases/main.go`, `.env.example`, `README.md`, `databases/README.md`: rollout non-superuser app-role доведён до fail-fast контракта — `DB_REQUIRE_RLS_EFFECTIVE=true` заставляет startup падать при superuser/BYPASSRLS, а `go run ./cmd/databases rls-status -require-effective` подходит для CI/deploy preflight.

**Валидация**

- [x] Тест-кейсы: запрет утечки данных между scope без явного разрешения
- [x] Дашборды или экспорт метрик задокументированы

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Сложность RLS | Начать с отдельных схем/БД на tenant; эволюция к RLS |

---

### v2.0 — Распределение, плагины, web-конфигурация

| | |
|--|--|
| **Цель** | Горизонтальное масштабирование шины, расширяемость (WASM/плагины), UI настройки scope/тегов «в пару кликов». |
| **Статус** | В работе. На 2026-04-13 foundation slice `v2.0` расширен до operator-ready surface: broker-контракт с consumer groups, versioned plugin registry с security-guardrails, воспроизводимый load scenario для broker foundation, web control plane с минимальным UX contract и отдельный Go-backed control-plane projection с endpoint-ами `/api/health`, `/api/scopes`, `/api/control-plane` и мутациями broker/module/plugin при сохранении frontend local fallback. Поверх этого закреплены общий JSON seed, file-backed snapshot, startup-гидратация plugin manifests и канонические demo manifests в репозитории, чтобы backend/frontend не расходились по дефолтам, операторские изменения переживали рестарт, а plugin slice мог опираться на реальные артефакты по умолчанию. |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| Распределённая шина | Брокер + контракты consumer groups |
| Плагины / WASM | Контракт загрузки, песочница, версионирование |
| Web-дашборд | Настройка модулей и контекстов без смены кода |

**Реализовано 2026-04-12**

- `core/distributed/broker.go`, `core/distributed/adapter.go`, `core/distributed/broker_test.go`: добавлен transport-agnostic broker contract для v2.0 с `Publish`, `SubscribeGroup`, delivery envelope, topic stats, round-robin consumer groups и adapter-ом `core/events.Event <-> distributed.Envelope`.
- `core/distributed/broker_test.go`: добавлен high-volume load scenario на 4096 publish-операций с несколькими consumer groups; тест валидирует fanout по группам, равномерный round-robin и отсутствие delivery loss/failures в baseline memory broker.
- `plugins/registry.go`, `plugins/registry_test.go`: проведён security-аудит plugin registry; manifest contract теперь режет absolute `entry`, неизвестные permissions и небезопасные `wasm+grpc` комбинации, сохраняя capability-based resolve и `LoadDir("*.plugin.json")`.
- `frontend/src/types/control-plane.ts`, `frontend/src/lib/api.ts`, `frontend/src/modules/control-plane/ControlPlaneDashboard.tsx`, `frontend/src/context/ScopeContext.tsx`, `frontend/src/test/control-plane.spec.tsx`, `frontend/tests/e2e/smoke.spec.ts`: web/PWA поверхность переведена на `Control Plane` с локально-персистентным snapshot-слоем для broker/module/plugin/scope настроек и проверкой через `npm run verify`.
- `frontend/src/modules/control-plane/ControlPlaneDashboard.tsx`, `frontend/src/test/control-plane.spec.tsx`, `frontend/README.md`: зафиксирован минимальный web UX contract для operator flow: first-screen status summary, именованные mutating actions, live event-trace feedback и тестовое подтверждение этих критериев.
- `controlplane/models.go`, `controlplane/service.go`, `controlplane/http.go`, `controlplane/*_test.go`, `cmd/controlplane/main.go`, `README.md`, `ARCHITECTURE.md`, `frontend/README.md`: добавлен Go-backed control-plane projection с in-memory snapshot service, HTTP/CORS surface для `/api/health`, `/api/scopes`, `/api/control-plane`, PATCH/POST мутациями broker/module/plugin и отдельным entrypoint `go run ./cmd/controlplane`; frontend сохраняет local fallback и может прозрачно переключаться на реальный backend.
- `controlplane/default_snapshot.json`, `controlplane/models.go`, `controlplane/default_snapshot_test.go`, `frontend/src/lib/api.ts`, `frontend/tsconfig.json`, `frontend/vite.config.ts`: дефолтный control-plane snapshot вынесен в общий JSON seed, так что Go projection и frontend fallback используют один и тот же broker/module/plugin/scope baseline без ручной синхронизации.
- `controlplane/service.go`, `controlplane/service_test.go`, `cmd/controlplane/main.go`, `README.md`, `frontend/README.md`: operator API получил file-backed snapshot; мутации пишутся атомарно в `CONTROL_PLANE_STATE_PATH` и переживают рестарт, а загрузка валидирует сохранённый state перед стартом HTTP surface.
- `controlplane/service.go`, `controlplane/service_test.go`, `cmd/controlplane/main.go`, `README.md`, `frontend/README.md`: plugin slice `control plane` теперь можно гидратировать из реальных `*.plugin.json` через `CONTROL_PLANE_PLUGIN_DIR`; manifest-данные обновляют version/runtime/protocol/entry/capabilities, а operator status и локальные overrides сохраняются из snapshot.
- `plugins/manifests/*.plugin.json`, `cmd/controlplane/main.go`, `README.md`, `frontend/README.md`: в репозиторий добавлены канонические demo manifests, а `cmd/controlplane` по умолчанию читает `plugins/manifests` как manifest source, поэтому plugin projection начинает работать на реальных артефактах без дополнительной настройки.

**Валидация**

- [x] Нагрузочный сценарий на брокер
- [x] Аудит безопасности плагинов
- [x] UX-критерии для web (минимальный набор)
- [x] Go-backed control-plane projection совместим с frontend API contract и покрыт unit/http тестами
- [x] Backend projection и frontend fallback используют единый seed-артефакт для operator snapshot
- [x] Control-plane mutations переживают рестарт backend-процесса и покрыты persistence-тестом
- [x] Plugin projection может гидратироваться из реальных manifests без потери operator status
- [x] В репозитории есть канонический manifest source для demo plugin slice по умолчанию

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| WASM сложность | Начать с отдельных OS-процессов и gRPC |
| Брокер как SPOF | Кластер, health-checks, DLQ |

---

## [Archived] — снимки фаз

_Пусто._ При смене крупных целей фаза копируется сюда с датой архивации и пометкой `[Archived]`, без удаления истории из основного текста выше (дублирование с пометкой «архивная копия»).

---

## Ответственность (RACI-кратко)

| Зона | Ответственный |
|------|----------------|
| ROADMAP / фазы | Lead Product Architect |
| `PROJECT_RULES.md` / gate | Tech Lead + Architect |
| Доменные модули | Владельцы пакетов (назначаются в PROJECT_RULES) |
| Релизы v1.0+ | Release Manager |

---

## Gate для ИИ и разработчиков

Если задача **не** отражена в ROADMAP под версией/контекстом:

1. Создать запись (Issue / ветка по правилам репозитория после появления трекера).
2. Добавить строку в соответствующую фазу и в Change Log.
3. Только после этого — реализация.

Пропуск проектирования, тестов или документации, если это **обязательно** по `PROJECT_RULES.md` — блокирующий; до принятия `PROJECT_RULES.md` — блок по **ARCHITECTURE.md** и чек-листам фазы.

---

## 📊 Change Log

| Дата | Что изменено | Почему | Кто утвердил |
|------|--------------|--------|--------------|
| 2026-04-10 | Создан `ROADMAP.md`: фазы v0.1–v2.0, версионирование, EXC-001 (нет PROJECT_RULES.md), Change Log, gate | Старт формальной продуктовой карты Modulr | Lead Product Architect |
| 2026-04-12 | `v1.5` дополнён runtime access-policy gate для auth-required dispatch и fallback diagnostics | Зафиксировать новый security slice без расхождения между transport auth и orchestrator | Lead Product Architect |
| 2026-04-12 | `v1.5` уточнён как почти завершённый кодовый этап; `v2.0` переведён в `В работе` и дополнен foundation slice (`core/distributed`, `plugins`, `frontend control plane`) | Зафиксировать фактическую точку продолжения разработки после последнего документированного v1.5 change-set | Lead Product Architect |
| 2026-04-12 | В `v2.0` закрыт validation-пункт по broker load scenario и уточнён статус foundation slice | Зафиксировать, что consumer-group baseline валидирован high-volume тестом, а не только smoke-сценарием | Lead Product Architect |
| 2026-04-12 | В `v2.0` закрыт validation-пункт по plugin security audit и уточнены guardrails manifest registry | Зафиксировать обязательные security-checks до появления реального plugin runner | Lead Product Architect |
| 2026-04-12 | В `v2.0` закрыт validation-пункт по минимальным web UX-критериям control plane | Зафиксировать operator-facing contract и привязать его к автоматическим тестам, а не к ручной визуальной проверке | Lead Product Architect |
| 2026-04-13 | `v2.0` дополнён Go-backed control-plane projection и validation-пунктом на совместимость frontend/backend API | Зафиксировать переход control plane из frontend-only snapshot режима в реальный операторский HTTP surface без ломки существующего fallback | Lead Product Architect |
| 2026-04-13 | `v2.0` дополнён общим JSON seed для control plane snapshot | Зафиксировать единый source of truth для Go projection и frontend fallback перед следующими operator-изменениями | Lead Product Architect |
| 2026-04-13 | `v2.0` дополнён file-backed control-plane state | Зафиксировать переход от ephemeral operator API к restart-safe конфигурации перед дальнейшей интеграцией с runtime registries | Lead Product Architect |
| 2026-04-13 | `v2.0` дополнён startup-гидратацией plugin manifests в control plane | Зафиксировать первый bridge между operator API и реальными plugin artifacts до появления полного plugin runner | Lead Product Architect |
| 2026-04-13 | `v2.0` дополнён каноническими demo plugin manifests в репозитории | Зафиксировать дефолтный manifest source для control plane, чтобы operator API работал на реальных артефактах без ручной подготовки окружения | Lead Product Architect |

---

📅 ROADMAP актуализирован. Следующий чек-ап: **2026-04-24**. Правила соблюдены.
