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

package abci

import (
	"context"

	"code.vegaprotocol.io/vega/core/txn"
	"github.com/tendermint/tendermint/abci/types"
)

//nolint:interfacebloat
type Tx interface {
	Command() txn.Command
	Unmarshal(interface{}) error
	PubKey() []byte
	PubKeyHex() string
	Party() string
	Hash() []byte
	Signature() []byte
	BlockHeight() uint64
	GetCmd() interface{}
	GetPoWNonce() uint64
	GetPoWTID() string
	GetVersion() uint32
}

type Codec interface {
	Decode(in []byte, chainID string) (Tx, error)
}

// ABCI hooks.
type (
	OnInitChainHandler        func(types.RequestInitChain) types.ResponseInitChain
	OnBeginBlockHandler       func(types.RequestBeginBlock) (context.Context, types.ResponseBeginBlock)
	OnEndBlockHandler         func(types.RequestEndBlock) (context.Context, types.ResponseEndBlock)
	OnCheckTxHandler          func(context.Context, types.RequestCheckTx, Tx) (context.Context, types.ResponseCheckTx)
	OnDeliverTxHandler        func(context.Context, types.RequestDeliverTx, Tx) (context.Context, types.ResponseDeliverTx)
	OnCommitHandler           func() types.ResponseCommit
	ListSnapshotsHandler      func(types.RequestListSnapshots) types.ResponseListSnapshots
	OffserSnapshotHandler     func(types.RequestOfferSnapshot) types.ResponseOfferSnapshot
	LoadSnapshotChunkHandler  func(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk
	ApplySnapshotChunkHandler func(context.Context, types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk
	InfoHandler               func(types.RequestInfo) types.ResponseInfo
	OnCheckTxSpamHandler      func(Tx) types.ResponseCheckTx
	OnDeliverTxSpamHandler    func(context.Context, Tx) types.ResponseDeliverTx
)
