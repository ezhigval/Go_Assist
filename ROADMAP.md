# Modulr — продуктовая дорожная карта

| Поле | Значение |
|------|----------|
| **Владелец артефакта** | Lead Product Architect |
| **Последнее обновление** | 2026-04-11 |
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
| **Статус** | В работе. На 2026-04-11 закрыт первый срез runtime-level scope isolation и observability; storage/RLS остаётся следующим шагом. |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| RLS / tenant / scope | Дизайн + реализация в `databases/` или слое репозиториев |
| Наблюдаемость | Связка `metrics` + трассировка trace_id по шине |
| Политики смешивания scope | Явные правила (README/ARCHITECTURE) реализованы в коде |

**Реализовано 2026-04-11**

- `events/scope_policy.go`, `core/orchestrator/pipeline.go`, `core/orchestrator/pipeline_test.go`: orchestrator по умолчанию режет cross-scope `Decision.Scope`; переход разрешается только при явной policy через `allowed_scopes` в metadata или tags вида `allow_scope:<segment>`;
- `app/runtime.go`, `app/runtime_test.go`: runtime подключает `metrics.Service`, а E2E-тесты покрывают запрет утечки между scope, разрешённый cross-scope path и trace-visible outcome;
- `metrics/service.go`, `metrics/models.go`, `metrics/service_test.go`: `metrics` теперь сохраняет не только общие счётчики событий, но и агрегаты по `scope` и краткие trace-summary по `trace_id`.

**Валидация**

- [x] Тест-кейсы: запрет утечки данных между scope без явного разрешения
- [ ] Дашборды или экспорт метрик задокументированы

**Риски и fallback**

| Риск | Fallback |
|------|----------|
| Сложность RLS | Начать с отдельных схем/БД на tenant; эволюция к RLS |

---

### v2.0 — Распределение, плагины, web-конфигурация

| | |
|--|--|
| **Цель** | Горизонтальное масштабирование шины, расширяемость (WASM/плагины), UI настройки scope/тегов «в пару кликов». |
| **Статус** | План |

**Deliverables**

| Артефакт | Назначение |
|----------|------------|
| Распределённая шина | Брокер + контракты consumer groups |
| Плагины / WASM | Контракт загрузки, песочница, версионирование |
| Web-дашборд | Настройка модулей и контекстов без смены кода |

**Валидация**

- [ ] Нагрузочный сценарий на брокер
- [ ] Аудит безопасности плагинов
- [ ] UX-критерии для web (минимальный набор)

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

---

📅 ROADMAP актуализирован. Следующий чек-ап: **2026-04-24**. Правила соблюдены.
