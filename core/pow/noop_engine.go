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

package pow

import (
	"code.vegaprotocol.io/vega/core/blockchain/abci"
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

func (e *NoopEngine) GetSpamStatistics(_ string) SpamStatistics {
	return SpamStatistics{}
}
