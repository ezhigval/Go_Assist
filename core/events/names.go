package events

// Распространённые имена событий ядра и транспорта.
const (
	V1MessageReceived             Name = "v1.message.received"
	V1OrchestratorActionDispatch  Name = "v1.orchestrator.action.dispatch"
	V1OrchestratorFallback        Name = "v1.orchestrator.fallback.requested"
	V1OrchestratorDecisionOutcome Name = "v1.orchestrator.decision.outcome"
	V1AIAnalyzeRequest            Name = "v1.ai.analyze.request"
	V1AIAnalyzeResult             Name = "v1.ai.analyze.result"
)
