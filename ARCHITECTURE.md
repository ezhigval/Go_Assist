# 🏗️ Архитектура Modulr

> Event-driven LEGO-платформа: **scope** (жизнь) × **operations** (действия), ядро и ИИ сверху, модули снизу.

---

## 🧠 Философия

1. **Контекст-агностика модулей** — внутри пакета нет ветвлений «только для pets». Разница — в **метаданных** (`scope`, `tags`, `owner_id` в перспективе) и в **политиках** ядра/хранилища.  
2. **Единый контракт** — например, создание транзакции одинаково по форме для `personal` и `business`; отличаются права, лимиты и маршрутизация.  
3. **Связи только через события** — `v1.{module}.{action}`; модули **не импортируют** друг друга.  
4. **ИИ как маршрутизатор** — текст/событие → интент → `[]Decision{Target, Action, Parameters, Confidence}` → проверка **реестра** → публикация на шину.  
5. **Обратная связь** — после исполнения ожидается поток `v1.{module}.{action}.completed` (или агрегированный outcome), чтобы **aiengine** обновлял веса и **metrics** накапливал KPI.  
6. **Изоляция данных** — по умолчанию данные одного `scope` не смешиваются с другим; пересечение только через явные теги, роли и подтверждение.

---

## 🧩 Слои

```
┌─────────────────────────────────────────────────────────────────┐
│  Transport: telegram · (web, cli — план)                         │
├─────────────────────────────────────────────────────────────────┤
│  Core: core/orchestrator · core/aiengine · core/events (шина ядра)│
├─────────────────────────────────────────────────────────────────┤
│  Domain bus: modulr/events (v1.*, scope, Storage, MemoryBus)    │
├─────────────────────────────────────────────────────────────────┤
│  Operations: finance, tracker, knowledge, metrics, notifications│
│              media, email, organizer*, auth, scheduler, files  │
├─────────────────────────────────────────────────────────────────┤
│  Data: databases (PostgreSQL), персистентность за интерфейсами   │
└─────────────────────────────────────────────────────────────────┘
```

\*`organizer` — композит календаря, todo, notes, contacts (операции calendar/tracker/knowledge/contacts).

---

## 🌍 Scope (жизненные контексты)

Канон объявлен в **`events/segment.go`**:

`personal` · `family` · `work` · `business` · `health` · `travel` · `pets` · `assets`

Вспомогательные функции: `IsValidSegment`, `ParseSegmentFromAny`, `IsCareerScope` (work+business для карьерных сценариев), `DefaultSegment()`.

Сущности доменных пакетов используют поле **`Context`** типа `Segment` в JSON как **life scope** (историческое имя поля — `context`).

Правило runtime-level изоляции: `Decision.Scope` не может бесшумно сменить исходный scope запроса. Cross-scope путь разрешается только явно:

- metadata `allowed_scopes: ["business", ...]`;
- или tag `allow_scope:<segment>`.

Иначе orchestrator отфильтрует решение до dispatch.

---

## 📡 Две шины (текущее состояние)

| Пакет | Назначение |
|-------|------------|
| **`modulr/events`** | Основная шина доменных модулей (`events.Bus`, `events.Event`, версии `v1.*`). |
| **`modulr/core/events`** | Шина ядра оркестратора (`EventBus`, события вроде `v1.message.received`, `v1.orchestrator.action.dispatch`). |

В одном процессе их связывает **`core/busbridge`**:

- domain → core: зеркалит доменные события в шину ядра, сохраняет `trace_id`, `chat_id`, `scope`, `tags`;
- core → domain: зеркалит события ядра обратно в `modulr/events`, поддерживает alias-маршруты для несовпадающих имён;
- защита от циклов: служебный маркер в `Context` не даёт событию бесконечно «пинг-понговать» между шинами.

`core/events.MemoryBus` поддерживает `SubscribeAll` для пассивных наблюдателей и адаптера, не меняя основной контракт `EventBus`.

---

## 🔄 Поток: сообщение → ИИ → модули

1. Транспорт публикует (на доменной или ядровой шине) событие с текстом и **`scope`**.  
2. **Orchestrator** обогащает scope, валидирует, вызывает **AIEngine.Analyze** (или эквивалент по шине).  
3. **AIEngine** выбирает модели (роутер), возвращает решения; оркестратор фильтрует по **confidence ≥ threshold** и **`Registry.HasEndpoint`**.  
4. Публикуется **`v1.orchestrator.action.dispatch`** и затем целевые **`v1.{module}.{action}`**.  
5. **Monitor** пишет метрики и историю по `chat_id`; модули шлют outcome → **Feedback** в ИИ.

Реализация: `core/orchestrator/*`, `core/aiengine/*`.

---

## 📦 Реестр модулей (оркестратор)

`core/orchestrator/registry.go` хранит допустимые пары `(module, action)`. Сид по умолчанию покрывает calendar, tracker, maps, knowledge, finance, contacts, metrics, logistics, inventory, notifications, media — чтобы цепочки из ТЗ и ИИ не указывали в пустоту. Расширение: **`RegisterModule`** на старте приложения.

---

## 🗄️ Данные

- **`events.Storage`** + **`MemoryStorage`** — универсальная JSON-персистентность для модулей без привязки к СУБД.  
- **`databases/`** — PostgreSQL для прод-сценариев с Telegram/сессиями, trace-связанным `event_journal`, scope-aware `stats` и DB-backed `auth_sessions`; repository-слой выставляет единый storage access context (`modulr.allowed_scopes`, `modulr.chat_id`, `modulr.auth_token_hash`, `modulr.scope_bypass`).  
- Политики **RLS / tenant / scope** уже покрывают `event_journal`, `stats`, `sessions` и `auth_sessions`; `telegram` использует DB-backed auth store через opaque `session reference`, а rollout по окружениям теперь опирается на `DB_REQUIRE_RLS_EFFECTIVE=true` и deploy preflight `rls-status -require-effective`.

---

## 🤖 AI

- **`core/aiengine`**: запросы, решения, фидбек, роутер моделей (LLM, route_planner, finance_analyzer, schedule_optimizer — как виды).  
- Реальный инференс помечен **`// TODO: Real Model Integration`**.  
- Сырые PII в долгоживущем виде в aiengine не хранятся — только метаданные решений и фидбека.

---

## 🛡️ Надёжность и безопасность

- Отмена через **`context.Context`**, блокировки **`sync.RWMutex`** в конкурентных структурах.  
- Паники обработчиков шины — лог + dead-letter (где реализовано).  
- При `auth_required` orchestrator режет доменные dispatch-решения без ролей или с ролью, которой target event не разрешён; summary причин идёт в fallback для диагностики.
- Fallback оркестратора: **`v1.orchestrator.fallback.requested`** при сбое ИИ или пустом допустимом наборе решений.

---

## 📈 Масштабирование

- Вертикально: пулы БД, кэш.  
- Горизонтально: вынести шину в broker contract (`core/distributed`) и раскладывать обработку по consumer groups / stateless-воркерам.  
- Расширяемость: versioned plugin manifests (`plugins/`) описывают runtime `process|wasm`, entrypoint, protocol и capability map до фактической загрузки sandbox/process runner; registry уже режет absolute entry paths, неизвестные permissions и небезопасные runtime/protocol комбинации.
- Продуктово: **настройка scope/тегов в UI** «в пару кликов» через web control plane без смены кода модулей.

---

## 📚 Справочник пакетов (ориентир)

| Пакет | Назначение |
|-------|------------|
| `app` | Runtime-сборка v0.3: входящее сообщение → core bus → orchestrator → bus bridge → доменные обработчики |
| `telegram`, `databases`, `auth` | Вход, данные, сессии, event journal |
| `core/distributed` | Foundation-контракт для broker-backed event lanes и consumer groups в v2.0 |
| `plugins` | Versioned plugin manifests и capability registry для process/WASM expansion path |
| `organizer` | Календарь, todo, заметки, контакты |
| `events`, `metrics`, `notifications`, `scheduler`, `files` | Инфраструктура и сквозные возможности |
| `finance`, `tracker`, `knowledge`, `media`, `email` | Доменные операции |
| `core/orchestrator`, `core/aiengine`, `core/events` | Мозг и ИИ-хаб |
| `frontend/src/modules/control-plane` | Web/PWA control plane для настройки broker/module/plugin/scope snapshot в dev/runtime UI |

---

## ⚠️ Золотое правило

> Доменный модуль **не** импортирует другой доменный модуль. Только **контракты**, **шина**, **scope/tags** и вызовы **ядра** там, где это явно разрешено архитектурой сборки.

Обзор продукта — [**README.md**](./README.md).
