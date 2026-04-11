# AI Architecture

## Components

- `core/aiengine` — routing, decision scoring, feedback weights.
- `ai/` — domain-facing AI suggestion module on top of `events.Bus`.
- `ai/docker-compose.local.yml` — local inference stack (`ollama` + `ai-router`).

## Request Flow

1. Input event enters orchestrator (`v1.message.received`).
2. Orchestrator builds `aiengine.Request` with `scope`, `tags`, `metadata`.
3. `core/aiengine` selects models and emits `[]Decision`.
4. Orchestrator validates decision against registry and confidence threshold.
5. Valid decisions are dispatched as `v1.{module}.{action}` events.
6. Execution outcomes return via `v1.orchestrator.decision.outcome`.

## Data Safety

- Scope is validated against `events/segment.go`.
- Raw PII is not persisted in AI state.
- Confidence threshold defaults to `0.7` in orchestrator.

## Local Mode

- Start local stack with:

```bash
cd ai
docker compose -f docker-compose.local.yml up -d
```

- Local router exposes:
  - `GET /health`
  - `POST /infer`
