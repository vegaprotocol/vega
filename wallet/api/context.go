package api

import "context"

func TraceIDFromContext(ctx context.Context) string {
	traceID := ctx.Value("trace-id")
	if traceID == nil {
		return ""
	}
	return traceID.(string)
}
