package events

import "context"

type traceKey struct{}

// WithTraceID кладёт trace id в context (вход бота/HTTP).
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceKey{}, traceID)
}

// TraceIDFromContext достаёт trace id (пустая строка, если нет).
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceKey{}).(string)
	return v
}
