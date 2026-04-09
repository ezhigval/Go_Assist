# Changelog Modulr

Формат записи аудита и крупных правок:

`- [YYYY-MM-DD] | путь/к файлу | суть изменения | правило PROJECT_RULES.md (раздел) или пометка`

Мелкие правки можно группировать одной строкой с перечислением файлов.

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
