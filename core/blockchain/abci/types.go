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
	OfferSnapshotHandler      func(types.RequestOfferSnapshot) types.ResponseOfferSnapshot
	LoadSnapshotChunkHandler  func(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk
	ApplySnapshotChunkHandler func(context.Context, types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk
	InfoHandler               func(types.RequestInfo) types.ResponseInfo
	OnCheckTxSpamHandler      func(Tx) types.ResponseCheckTx
	OnDeliverTxSpamHandler    func(context.Context, Tx) types.ResponseDeliverTx
)
