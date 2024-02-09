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
	GetNonce() uint64
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
	OfferSnapshotHandler      func(context.Context, *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error)
	LoadSnapshotChunkHandler  func(context.Context, *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkHandler func(context.Context, *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error)
	InfoHandler               func(context.Context, *types.RequestInfo) (*types.ResponseInfo, error)
	OnCheckTxSpamHandler      func(Tx) types.ResponseCheckTx
	FinalizeHandler           func() []byte
)
