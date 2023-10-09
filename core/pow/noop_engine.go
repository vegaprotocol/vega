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

package pow

import (
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/libs/crypto"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"github.com/ethereum/go-ethereum/common/math"
)

type NoopEngine struct {
	blockHeight uint64
	blockHash   string
}

func NewNoop() *NoopEngine {
	return &NoopEngine{}
}

func (e *NoopEngine) BeginBlock(blockHeight uint64, blockHash string) {
	e.blockHeight = blockHeight
	e.blockHash = blockHash
}

func (e *NoopEngine) EndOfBlock() {}
func (e *NoopEngine) Commit()     {}
func (e *NoopEngine) CheckTx(tx abci.Tx) error {
	return nil
}

func (e *NoopEngine) DeliverTx(tx abci.Tx) error {
	return nil
}

func (e *NoopEngine) IsReady() bool                     { return true }
func (e *NoopEngine) SpamPoWNumberOfPastBlocks() uint32 { return uint32(0) }
func (e *NoopEngine) SpamPoWDifficulty() uint32         { return uint32(0) }
func (e *NoopEngine) SpamPoWHashFunction() string       { return "" }
func (e *NoopEngine) SpamPoWNumberOfTxPerBlock() uint32 { return uint32(0) }
func (e *NoopEngine) SpamPoWIncreasingDifficulty() bool { return false }

func (e *NoopEngine) BlockData() (uint64, string) {
	return e.blockHeight, e.blockHash
}

func (e *NoopEngine) GetSpamStatistics(_ string) *protoapi.PoWStatistic {
	var expected uint64
	return &protoapi.PoWStatistic{
		NumberOfPastBlocks: 500,
		BlockStates: []*protoapi.PoWBlockState{
			{
				BlockHeight:          e.blockHeight,
				BlockHash:            e.blockHash,
				TransactionsSeen:     0,
				ExpectedDifficulty:   &expected,
				HashFunction:         crypto.Sha3,
				Difficulty:           0,
				TxPerBlock:           math.MaxUint64,
				IncreasingDifficulty: true,
			},
		},
	}
}
