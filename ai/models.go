package ai

// Suggestion предложение действия (ИИ только предлагает; подтверждение — правила/пользователь).
type Suggestion struct {
	TargetModule string
	Action       string
	Reason       string
	Confidence   float64
	Payload      any
}
