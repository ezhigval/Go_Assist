# Modulr Release Template

Шаблон для релизов `v1.0+`. Используется для release notes, go-live checklist и rollback decision log.

## 1. Release Summary

- Version / tag:
- Date:
- Owner:
- Scope:
- Related PRs / commits:

## 2. Contract Impact

- New `v1.*` events:
- Deprecated events:
- Breaking changes:
- Compatibility window:
- Updated docs:

Если breaking changes отсутствуют, явно указать: `No breaking API/event changes`.

## 3. Migrations

- Migration step:
- Command:

```bash
DB_AUTO_MIGRATE=false go run ./databases/cmd/databases up
go run ./databases/cmd/databases status
```

- Expected versions:
- Backfill / manual steps:
- Irreversible changes:

## 4. Runtime / AI Flags

- Image / artifact version:
- `AI_PROVIDER`: `stub` | `local`
- `AI_PROVIDER_BASE_URL`:
- `AI_ALLOW_STUB_FALLBACK`:
- `AI_PROVIDER_TIMEOUT_MS`:
- `DB_AUTO_MIGRATE=false` confirmed:
- `TELEGRAM_STATE_STORE`:

Если релиз включает новый provider path, указать:

- health endpoint:
- fallback expectation:
- degraded mode behaviour:

## 5. Validation Before Go-Live

- [ ] `./scripts/modulr-check.sh`
- [ ] `frontend/npm run build` (если затронут frontend)
- [ ] `go run ./databases/cmd/databases status` показывает ожидаемые версии
- [ ] transport smoke: `telegram -> runtime -> outcome/fallback`
- [ ] AI smoke для выбранного `AI_PROVIDER`
- [ ] release notes обновлены

## 6. Rollback Plan

Rollback order:

1. Остановить новые writers / consumers, которые зависят от релиза.
2. Если релиз включал новый AI path, переключить:

```bash
AI_PROVIDER=stub
AI_ALLOW_STUB_FALLBACK=true
```

3. Переключить runtime на предыдущую версию образа / бинаря.
4. Если релиз включал additive migration, оставить схему и откатить только приложение.
5. Если релиз включал incompatible migration, использовать один из вариантов:
   - `go run ./databases/cmd/databases down -steps=N`, если down-step заранее проверен;
   - восстановление из snapshot/backup, если миграция необратима или data-destructive.
6. Повторить smoke-check на предыдущей версии.

Rollback decision log:

- Trigger:
- Time:
- Operator:
- Chosen path:
- Recovery verification:

## 7. Post-Release Notes

- Observed issues:
- Follow-up tasks:
- Metrics / traces reviewed:
- Changelog updated:
