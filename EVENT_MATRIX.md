# Modulr Event Matrix v1

Канонический артефакт для текущих стабильных контрактов `v1.*` в основном runtime path.

Актуально на: `2026-04-11`

## Scope документа

Матрица ниже покрывает:

- transport -> core -> orchestrator -> action runtime path, который реально подключён в `app/`, `telegram/`, `core/`;
- доменные события модулей, публикуемые текущими сервисами `finance`, `tracker`, `knowledge`, `email`, `media`;
- alias `core -> domain`, которые уже поддерживаются `core/busbridge`.

Документ не обещает стабильность для событий, которые пока только объявлены в коде, но не публикуются в поддерживаемом сценарии.

## Правила совместимости `v1.*`

1. Имя события внутри мажорной ветки неизменно. Переименование = новый контракт (`v2.*` или новый `v1.*` + deprecation-cycle).
2. Новые поля payload/context добавляются только как optional.
3. Consumer обязан игнорировать неизвестные поля.
4. Удаление обязательного поля или смена его смысла недопустимы в `v1.*`.
5. `trace_id` должен проходить через весь runtime path и оставаться стабильным для одного пользовательского запроса.
6. `scope` передаётся отдельным полем core event или через domain `context/segment`; смешивание scope без явной политики запрещено.
7. Если payload события ещё не строго типизирован структурой, стабильным считается только набор полей, явно перечисленный в этой матрице.

## Core / Transport Contracts

| Event | Producer | Consumer | Stable fields |
|------|----------|----------|---------------|
| `v1.message.received` | transport (`app/runtime`, `telegram`) | `core/orchestrator` | payload: `text`; core envelope: `chat_id`, `scope`, `tags[]`; context: `trace_id`, `source`, `user_id`, `username` |
| `v1.orchestrator.action.dispatch` | `core/orchestrator/pipeline` | observability, journal, passive listeners | payload: `decisions[]`, `trace_id`; envelope: `chat_id`, `scope`, `tags=["orchestrator","dispatch"]` |
| `v1.orchestrator.decision.outcome` | `app/actions.go` | `app/runtime`, transport response flow, journal | payload: `model_id`, `decision_id`, `action_event`, `ok`, `error`, `latency_ms`, `scope`; context: `trace_id`, `scope`, `chat_id` |
| `v1.orchestrator.fallback.requested` | `core/orchestrator` | `app/runtime`, transport fallback UX, future operator flows | payload: `original`, `reason`, `suggestion`; envelope: `chat_id`, `scope`; context: `trace_id` |
| `v1.transport.response.timeout` | `app/runtime` | journal / diagnostics | payload: `error`; metadata/context sidecar may include transport source |

Reserved, но пока не публикуются в основном runtime path:

- `v1.ai.analyze.request`
- `v1.ai.analyze.result`

## Runtime Dispatch Contracts

| Event | Producer | Consumer | Stable fields |
|------|----------|----------|---------------|
| `v1.calendar.create_event` | `core/orchestrator/pipeline` | `core/busbridge` alias -> `v1.calendar.created` | payload: `title`; optional: `start`, `context`; если `start/context` не заданы, runtime alias добавляет их автоматически |
| `v1.tracker.create_reminder` | `core/orchestrator/pipeline` | `app/actions.go` -> `tracker.Service.AddChecklistItem` | payload: хотя бы одно из `title|note|text|query`; optional: `due_at|due`, `tags[]`; context: `decision_id`, `model_id`, `chat_id`, `scope` |
| `v1.tracker.create_task` | `core/orchestrator/pipeline` | `app/actions.go` -> `tracker.Service.AddChecklistItem` | те же поля, что у `v1.tracker.create_reminder` |
| `v1.finance.create_transaction` | `core/orchestrator/pipeline` | `app/actions.go` -> `finance.Service.CreateTransaction` | optional payload fields: `type`, `amount_minor`, `currency`, `category`, `counterparty`, `memo|title|text|note`, `linked_entity_ids[]`, `tags[]`; context: `decision_id`, `model_id`, `chat_id`, `scope` |
| `v1.knowledge.save_query` | `core/orchestrator/pipeline` | `app/actions.go` -> `knowledge.Service.SaveArticle` | payload: хотя бы одно из `text|note|query|body`; optional: `title`, `topics[]`, `tags[]`; context: `decision_id`, `model_id`, `chat_id`, `scope` |
| `v1.knowledge.save_note` | `core/orchestrator/pipeline` | `app/actions.go` -> `knowledge.Service.SaveArticle` | те же поля, что у `v1.knowledge.save_query` |

## Domain Events In Current Runtime

| Event | Producer | Consumer | Stable fields |
|------|----------|----------|---------------|
| `v1.calendar.created` | `core/busbridge` alias для `v1.calendar.create_event` | calendar / organizer listeners | payload: `title`, `start`, `context` |
| `v1.finance.transaction.created` | `finance.Service.CreateTransaction` | observers / future analytics | payload: `finance.Transaction` |
| `v1.finance.budget.exceeded` | `finance.Service.recheckBudget` | alerts / notifications / reporting | payload: `segment`, `category`, `spent`, `limit` |
| `v1.tracker.plan.created` | `tracker.Service.CreatePlan` | observers / reporting | payload: `tracker.Plan`; context: `segment` |
| `v1.tracker.milestone.reached` | `tracker.Service.ReachMilestone` | observers / metrics / notifications | payload: `plan_id`, `title`, `context`, `tags`, `milestone` |
| `v1.tracker.habit.logged` | `tracker.Service.LogHabit` | observers / metrics | payload: `habit_id`, `streak`, `at` |
| `v1.reminder.on_route` | `tracker.Service.onNoteCreated` | future geo/reminder flows | payload: `hint`, `context`, `lat`, `lon`, `note_id`, `geofence_km` |
| `v1.knowledge.saved` | `knowledge.Service.SaveArticle` | observers / recommendation flow | payload: `knowledge.Article` |
| `v1.knowledge.recommendation` | `knowledge.Service.recommendRelated` | recommendation consumers / UI hints | payload: `base_article_id`, `related_ids[]` |
| `v1.email.received` | `email.Service.IngestIncoming` | rules / workflow consumers | payload: `id`, `subject`, `body`, `context`, `tags[]`, `from`, `message_id`; context: `segment` |
| `v1.email.action_required` | `email.Service` calendar bridge logic | workflow / notification consumers | payload: `kind`, `title`, `start`, `trace`, `context` |
| `v1.email.sent` | `email.Service.SendOutgoing` | observers / audit | payload: `id`, `subject`, `body`, `context`, `tags[]`, `from`, `message_id` |
| `v1.media.uploaded` | `media.Service.Upload` | media observers / indexing | payload: `media.MediaItem`; context: `segment` |
| `v1.media.linked` | `media.Service.Link` | graph / relation consumers | payload: `media_id`, `entity_type`, `entity_id` |
| `v1.media.tagged` | `media.Service.AddTags` | search / indexing consumers | payload: `media_id`, `tags[]` |

## Notes On Versioning

- Если runtime начнёт эмитить `v1.ai.analyze.request/result`, сначала обновить эту матрицу и только затем считать их частью стабильного контракта.
- Если `v1.finance.create_transaction` потребует строгий mandatory payload (например, `amount_minor > 0`), это нужно выпускать как новый контракт или с backward-compatible fallback.
- Любой новый consumer должен опираться на `trace_id` и не должен требовать отсутствия неизвестных полей.
