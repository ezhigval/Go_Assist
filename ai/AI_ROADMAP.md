# AI Roadmap

## v0.1

- Stub gateway and deterministic inference paths.
- Feedback loop wiring in `core/aiengine`.

## v0.2

- Provider health checks in background loop.
- Configurable provider list (remote + local) in YAML.
- Structured decision telemetry for metrics.

## v0.3

- Real provider integration (OpenAI/Ollama) behind interfaces.
- PII redaction middleware before external calls.
- Contract tests for `Request -> Decision` mapping.

## v1.0

- Hybrid routing with policy per scope (`personal`, `business`, etc.).
- Confidence calibration by module/action.
- Automatic failover to local inference on provider outage.
