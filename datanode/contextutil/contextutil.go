// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package contextutil

import (
	"context"
	"errors"

	uuid "github.com/satori/go.uuid"
)

type (
	remoteIPAddrKey int
	traceIDT        int
	blockHeight     int
)

var (
	clientRemoteIPAddrKey remoteIPAddrKey
	traceIDKey            traceIDT
	blockHeightKey        blockHeight

	ErrBlockHeightMissing = errors.New("no or invalid block height set on context")
)

// WithRemoteIPAddr wrap the context into a new context
// and embed the ip addr as a key.
func WithRemoteIPAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, clientRemoteIPAddrKey, addr)
}

// RemoteIPAddrFromContext returns the remote IP addr value stored in ctx, if any.
func RemoteIPAddrFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(clientRemoteIPAddrKey).(string)
	return u, ok
}

// TraceIDFromContext get traceID from context (add one if none is set).
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

// WithTraceID returns a context with a traceID value.
func WithTraceID(ctx context.Context, tID string) context.Context {
	return context.WithValue(ctx, traceIDKey, tID)
}

func WithBlockHeight(ctx context.Context, h int64) context.Context {
	return context.WithValue(ctx, blockHeightKey, h)
}
