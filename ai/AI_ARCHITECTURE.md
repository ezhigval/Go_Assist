# AI Architecture

## Components

- `core/aiengine` — routing, decision scoring, feedback weights.
- `ai/` — domain-facing AI suggestion module on top of `events.Bus`.
- `ai/docker-compose.local.yml` — local inference stack (`ollama` + `ai-router`).
- `core/aiengine/provider.go` — gated external provider path (`AI_PROVIDER=local`) с stub fallback.

## Request Flow

1. Input event enters orchestrator (`v1.message.received`).
2. Orchestrator builds `aiengine.Request` with `scope`, `tags`, `metadata`.
3. `core/aiengine` selects models and emits `[]Decision`.
   If `AI_PROVIDER=local`, decisions are requested from `POST /infer`; if provider fails and `AI_ALLOW_STUB_FALLBACK=true`, engine falls back to deterministic stub inference.
4. Orchestrator validates decision against registry and confidence threshold.
5. Valid decisions are dispatched as `v1.{module}.{action}` events.
6. Execution outcomes return via `v1.orchestrator.decision.outcome`.

## Data Safety

- Scope is validated against `events/segment.go`.
- Raw PII is not persisted in AI state.
- Before external provider calls, request text is redacted for basic email/phone/card patterns.
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

- Runtime env for gated integration:

```bash
AI_PROVIDER=local
AI_PROVIDER_BASE_URL=http://127.0.0.1:8000
AI_ALLOW_STUB_FALLBACK=true
AI_PROVIDER_TIMEOUT_MS=2500
```
