package pow

import (
	"code.vegaprotocol.io/vega/blockchain/abci"
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
