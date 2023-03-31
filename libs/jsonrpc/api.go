package jsonrpc

import "context"

const (
	VERSION2   string  = "2.0"
	TraceIDKey TraceID = "trace-id"
)

type TraceID string

func TraceIDFromContext(ctx context.Context) string {
	rawTraceID := ctx.Value(TraceIDKey)
	traceID := rawTraceID.(string)
	return traceID
}
