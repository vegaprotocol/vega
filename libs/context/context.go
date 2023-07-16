// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package context

import (
	"context"
	"errors"
	"strings"

	uuid "github.com/satori/go.uuid"
)

type (
	remoteIPAddrKey int
	traceIDT        int
	blockHeight     int
	chainID         int
	txHash          int
)

var (
	clientRemoteIPAddrKey remoteIPAddrKey
	traceIDKey            traceIDT
	blockHeightKey        blockHeight
	chainIDKey            chainID
	txHashKey             txHash

	ErrBlockHeightMissing = errors.New("no or invalid block height set on context")
	ErrChainIDMissing     = errors.New("no or invalid chain id set on context")
	ErrTxHashMissing      = errors.New("no or invalid transaction hash set on context")
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

func BlockHeightFromContext(ctx context.Context) (uint64, error) {
	hv := ctx.Value(blockHeightKey)
	if hv == nil {
		return 0, ErrBlockHeightMissing
	}
	h, ok := hv.(uint64)
	if !ok {
		return 0, ErrBlockHeightMissing
	}
	return h, nil
}

func ChainIDFromContext(ctx context.Context) (string, error) {
	cv := ctx.Value(chainIDKey)
	if cv == nil {
		return "", ErrChainIDMissing
	}
	c, ok := cv.(string)
	if !ok {
		return "", ErrChainIDMissing
	}
	return c, nil
}

func TxHashFromContext(ctx context.Context) (string, error) {
	cv := ctx.Value(txHashKey)
	if cv == nil {
		// if this is not happening in the context of a transaction, use the hash of the block
		cv = ctx.Value(traceIDKey)
		if cv == nil {
			return "", ErrTxHashMissing
		}
	}
	c, ok := cv.(string)
	if !ok {
		return "", ErrTxHashMissing
	}
	return c, nil
}

// WithTraceID returns a context with a traceID value.
func WithTraceID(ctx context.Context, tID string) context.Context {
	tID = strings.ToUpper(tID)
	return context.WithValue(ctx, traceIDKey, tID)
}

func WithBlockHeight(ctx context.Context, h uint64) context.Context {
	return context.WithValue(ctx, blockHeightKey, h)
}

func WithChainID(ctx context.Context, chainID string) context.Context {
	return context.WithValue(ctx, chainIDKey, chainID)
}

func WithTxHash(ctx context.Context, txHash string) context.Context {
	txHash = strings.ToUpper(txHash)
	return context.WithValue(ctx, txHashKey, txHash)
}
