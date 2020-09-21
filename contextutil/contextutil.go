package contextutil

import (
	"context"
	"errors"

	uuid "github.com/satori/go.uuid"
)

type remoteIPAddrKey int
type traceIDT int
type commandID int

type blockHeight int

var (
	clientRemoteIPAddrKey remoteIPAddrKey
	traceIDKey            traceIDT
	blockHeightKey        blockHeight
	commandIDKey          commandID

	ErrBlockHeightMissing = errors.New("no or invalid block height set on context")
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
		if h, _ := BlockHeightFromContext(ctx); h == 0 {
			ctx = context.WithValue(ctx, traceIDKey, "genesis")
			return ctx, "genesis"
		}
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

func BlockHeightFromContext(ctx context.Context) (int64, error) {
	hv := ctx.Value(blockHeightKey)
	if hv == nil {
		return 0, ErrBlockHeightMissing
	}
	h, ok := hv.(int64)
	if !ok {
		return 0, ErrBlockHeightMissing
	}
	return h, nil
}

// WithTraceID returns a context with a traceID value
func WithTraceID(ctx context.Context, tID string) context.Context {
	return context.WithValue(ctx, traceIDKey, tID)
}

func WithBlockHeight(ctx context.Context, h int64) context.Context {
	return context.WithValue(ctx, blockHeightKey, h)
}

func WithCommandID(ctx context.Context, cID string) context.Context {
	return context.WithValue(ctx, commandIDKey, cID)
}

func CommandIDFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(commandIDKey).(string)
	return u, ok
}
