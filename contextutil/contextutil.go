package contextutil

import (
	"context"

	uuid "github.com/satori/go.uuid"
)

type remoteIPAddrKey int

type traceIDT int

var (
	clientRemoteIPAddrKey remoteIPAddrKey
	traceIDKey            traceIDT
)

// WithRemoteIPAddr wrap the context into a new context
// and embed the ip addr as a key
func WithRemoteIPAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, clientRemoteIPAddrKey, addr)
}

// RemoteIPAddrFromContext returns the remote IP addr value stored in ctx, if any.
func RemoteIPAddrFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(clientRemoteIPAddrKey).(string)
	return u, ok
}

// TraceIDFromContext get traceID from context (add one if none is set)
func TraceIDFromContext(ctx context.Context) (context.Context, string) {
	tID := ctx.Value(traceIDKey)
	if tID == nil {
		stID := uuid.NewV4().String()
		ctx = context.WithValue(ctx, traceIDKey, stID)
		return ctx, stID
	}
	stID, ok := tID.(string)
	if !ok {
		stID = uuid.NewV4().String()
		ctx = context.WithValue(ctx, traceIDKey, stID)
	}
	return ctx, stID
}

// WithTraceID returns a context with a traceID value
func WithTraceID(ctx context.Context, tID string) context.Context {
	return context.WithValue(ctx, traceIDKey, tID)
}
