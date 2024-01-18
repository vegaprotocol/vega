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
	snapshot        int
)

var (
	clientRemoteIPAddrKey remoteIPAddrKey
	traceIDKey            traceIDT
	blockHeightKey        blockHeight
	chainIDKey            chainID
	txHashKey             txHash
	snapshotKey           snapshot

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

type snapshotInfo struct {
	version string
	upgrade bool
}

func WithSnapshotInfo(ctx context.Context, version string, upgrade bool) context.Context {
	return context.WithValue(ctx, snapshotKey, snapshotInfo{version: version, upgrade: upgrade})
}

// InProgressSnapshotRestore returns whether the data in the contexts tells us that a
// snapshot restore is in progress.
func InProgressSnapshotRestore(ctx context.Context) bool {
	v := ctx.Value(snapshotKey)
	if v == nil {
		return false
	}
	_, ok := v.(snapshotInfo)
	return ok
}

// InProgressUpgradeFrom returns whether the data in the contexts tells us that the
// node is restoring from a snapshot taken for a protocol-upgrade by the given version.
func InProgressUpgradeFrom(ctx context.Context, from string) bool {
	v := ctx.Value(snapshotKey)
	if v == nil {
		return false
	}
	si, ok := v.(snapshotInfo)
	if !ok {
		return false
	}
	return from == si.version && si.upgrade
}
