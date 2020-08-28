package stats

import (
	"context"
	"fmt"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"code.vegaprotocol.io/vega/blockchain/tm"
)

const (
	// Maximum sample size for average calculation, used in statistics (average tx per block etc).
	statsSampleSize = 5000
)

// Tendermint is a stats client for tendermint
type Tendermint struct {
	// construction params
	clt *tm.Client

	// state
	bc       *Blockchain
	txSizes  []int
	txTotals []uint64
}

func NewTendermint(clt *tm.Client) *Tendermint {
	return &Tendermint{
		clt: clt,
		bc:  NewBlockchain(),
	}
}

func (tm *Tendermint) Collect(ctx context.Context) error {
	fn := func(v tmctypes.ResultEvent) error {
		switch t := v.Data.(type) {
		case tmtypes.EventDataNewBlock:
			return tm.handleNewBlock(t)
		case tmtypes.EventDataTx:
			return tm.handleTx(t)
		default:
			return fmt.Errorf("don't know how to handle a %T", t)
		}
	}

	return tm.clt.Subscribe(ctx, fn,
		`tm.event = 'Tx'`,
		`tm.event = 'NewBlock'`,
	)
}

func (tm *Tendermint) handleNewBlock(e tmtypes.EventDataNewBlock) error {
	tm.bc.IncHeight()
	tm.setBatchStats()

	return nil
}

func (tm *Tendermint) handleTx(e tmtypes.EventDataTx) error {
	txLength := len(e.Tx)
	tm.setTxStats(txLength)

	return nil
}

// setBatchStats is used to calculate any statistics that should be
// recorded once per batch, typically called from commit.
func (tm *Tendermint) setBatchStats() {
	// Calculate the average total txn per batch, over n blocks
	if tm.txTotals == nil {
		tm.txTotals = make([]uint64, 0)
	}
	tm.txTotals = append(tm.txTotals, tm.bc.TotalTxLastBatch())
	totalTx := uint64(0)
	for _, itx := range tm.txTotals {
		totalTx += itx
	}
	averageTxTotal := totalTx / uint64(len(tm.txTotals))

	/*
		a.log.Debug("Batch stats for height",
			logging.Uint64("height", a.stats.Height()),
			logging.Uint64("average-tx-total", averageTxTotal))
	*/

	tm.bc.SetAverageTxPerBatch(averageTxTotal)
	tm.bc.SetTotalTxLastBatch(tm.bc.TotalTxCurrentBatch())
	tm.bc.SetTotalTxCurrentBatch(0)

	// MAX sample size for avg calculation is defined as const.
	if len(tm.txTotals) == statsSampleSize {
		tm.txTotals = nil
	}
}

func (tm *Tendermint) setTxStats(txLength int) {
	tm.bc.IncTotalTxCurrentBatch()
	if tm.txSizes == nil {
		tm.txSizes = make([]int, 0)
	}
	tm.txSizes = append(tm.txSizes, txLength)
	totalTx := 0
	for _, itx := range tm.txSizes {
		totalTx += itx
	}
	averageTxBytes := totalTx / len(tm.txSizes)

	/*
		tm.log.Debug("Transaction stats for height",
			logging.Uint64("height", tm.stats.Height()),
			logging.Int("average-tx-bytes", averageTxBytes))
	*/

	tm.bc.SetAverageTxSizeBytes(uint64(averageTxBytes))

	// MAX sample size for avg calculation is defined as const.
	if len(tm.txSizes) == statsSampleSize {
		tm.txSizes = nil
	}
}
