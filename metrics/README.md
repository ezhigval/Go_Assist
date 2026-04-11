# Metrics Module

`metrics/` — пассивный observability-слой для runtime и доменной шины.

## Что собирает

- общие счётчики событий (`Counts`);
- агрегаты по `scope` (`ScopeCounts`);
- краткие trace-summary по `trace_id` (`Snapshot`).

## Trace Summary

Каждый `TraceSummary` содержит:

- `trace_id`;
- последний известный `scope`;
- число событий в trace;
- последнее событие и его `source`;
- время последнего обновления.

Это даёт быстрый runtime-срез без чтения полного `event_journal`.

## Использование

```go
snapshot := runtime.Metrics().Snapshot(10)
log.Printf("scope_counts=%v traces=%v", snapshot.ScopeCounts, snapshot.Traces)
```

## Границы ответственности

- `metrics/` не подменяет `databases/event_journal`: journal хранит полную историю для replay/аудита;
- `metrics/` хранит только агрегаты и краткие trace-summary в памяти процесса;
- для production dashboard следующий шаг roadmap — экспорт этого snapshot в HTTP/Prometheus/OpenTelemetry слой.
