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
	"time"

	"code.vegaprotocol.io/vega/core/txn"
	"github.com/cometbft/cometbft/abci/types"
	types1 "github.com/cometbft/cometbft/proto/tendermint/types"
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
	GetLength() int
}

type Codec interface {
	Decode(in []byte, chainID string) (Tx, error)
}

// ABCI hooks.
type (
	PrepareProposalHandler    func(txs []Tx, raWtxs [][]byte) [][]byte
	ProcessProposalHandler    func(txs []Tx) bool
	OnInitChainHandler        func(*types.RequestInitChain) (*types.ResponseInitChain, error)
	OnBeginBlockHandler       func(uint64, string, time.Time, string, []Tx) context.Context
	OnEndBlockHandler         func(blockHeight uint64) (types.ValidatorUpdates, types1.ConsensusParams)
	OnCheckTxHandler          func(context.Context, *types.RequestCheckTx, Tx) (context.Context, *types.ResponseCheckTx)
	OnDeliverTxHandler        func(context.Context, Tx)
	OnCommitHandler           func() (*types.ResponseCommit, error)
	ListSnapshotsHandler      func(context.Context, *types.RequestListSnapshots) (*types.ResponseListSnapshots, error)
	OffserSnapshotHandler     func(context.Context, *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error)
	LoadSnapshotChunkHandler  func(context.Context, *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkHandler func(context.Context, *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error)
	InfoHandler               func(context.Context, *types.RequestInfo) (*types.ResponseInfo, error)
	OnCheckTxSpamHandler      func(Tx) types.ResponseCheckTx
	FinalizeHandler           func() []byte
)
