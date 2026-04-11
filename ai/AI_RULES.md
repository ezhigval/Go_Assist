# AI Rules

## 1. Contracts

- Input is `core/aiengine.Request`.
- Output is `[]core/aiengine.Decision`.
- Decisions must include `target`, `action`, `confidence`, `scope`.

## 2. Scope Safety

- `scope` must be one of `events.AllSegments()`.
- Cross-scope actions require explicit policy and audit trail.

## 3. Confidence Policy

- Orchestrator dispatch threshold defaults to `0.7`.
- `confidence < threshold` must result in fallback path.

## 4. Privacy

- Do not persist raw user text in long-lived AI storage.
- Do not log secrets, tokens, or user credentials.

## 5. Reliability

- Provider failures must not crash orchestrator.
- Every AI failure path must emit fallback signal.
- Feedback events must be idempotent by `decision_id`.
