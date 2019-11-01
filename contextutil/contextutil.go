package contextutil

import "context"

type remoteIPAddrKey int

var clientRemoteIPAddrKey remoteIPAddrKey

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
