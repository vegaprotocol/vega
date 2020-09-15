package processor

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
)

const (
	// Maximum sample size for average calculation, used in statistics (average tx per block etc).
	statsSampleSize = 5000
)

// setBatchStats is used to calculate any statistics that should be
// recorded once per batch, typically called from commit.
func (app *App) setBatchStats() {
	// Calculate the average total txn per batch, over n blocks
	app.txTotals = append(app.txTotals, app.stats.TotalTxLastBatch())
	totalTx := uint64(0)
	for _, itx := range app.txTotals {
		totalTx += itx
	}
	averageTxTotal := totalTx / uint64(len(app.txTotals))

	app.stats.SetAverageTxPerBatch(averageTxTotal)
	app.stats.SetTotalTxLastBatch(app.stats.TotalTxCurrentBatch())
	app.stats.SetTotalTxCurrentBatch(0)

	// MAX sample size for avg calculation is defined as const.
	if len(app.txTotals) == statsSampleSize {
		app.txTotals = app.txTotals[:0]
	}
}

func (app *App) updateStats() {
	app.stats.IncTotalBatches()
	avg := app.stats.TotalOrders() / app.stats.TotalBatches()
	app.stats.SetAverageOrdersPerBatch(avg)
	duration := time.Duration(app.currentTimestamp.UnixNano() - app.previousTimestamp.UnixNano()).Seconds()
	var (
		currentOrders, currentTrades uint64
	)
	app.stats.SetBlockDuration(uint64(duration * float64(time.Second.Nanoseconds())))
	if duration > 0 {
		currentOrders, currentTrades = uint64(float64(app.stats.CurrentOrdersInBatch())/duration),
			uint64(float64(app.stats.CurrentTradesInBatch())/duration)
	}
	app.stats.SetOrdersPerSecond(currentOrders)
	app.stats.SetTradesPerSecond(currentTrades)
	// log stats
	app.log.Debug("Processor batch stats",
		logging.Int64("previousTimestamp", app.previousTimestamp.UnixNano()),
		logging.Int64("currentTimestamp", app.currentTimestamp.UnixNano()),
		logging.Float64("duration", duration),
		logging.Uint64("currentOrdersInBatch", app.stats.CurrentOrdersInBatch()),
		logging.Uint64("currentTradesInBatch", app.stats.CurrentTradesInBatch()),
		logging.Uint64("total-batches", app.stats.TotalBatches()),
		logging.Uint64("avg-orders-batch", avg),
		logging.Uint64("orders-per-sec", currentOrders),
		logging.Uint64("trades-per-sec", currentTrades),
	)
	app.stats.NewBatch() // sets previous batch orders/trades to current, zeroes current tally
}

func (app *App) setTxStats(txLength int) {
	app.stats.IncTotalTxCurrentBatch()
	app.txSizes = append(app.txSizes, txLength)
	totalTx := 0
	for _, itx := range app.txSizes {
		totalTx += itx
	}
	averageTxBytes := totalTx / len(app.txSizes)

	app.log.Debug("Transaction stats for height",
		logging.Uint64("height", app.stats.Height()),
		logging.Int("average-tx-bytes", averageTxBytes))

	app.stats.SetAverageTxSizeBytes(uint64(averageTxBytes))

	// MAX sample size for avg calculation is defined as const.
	if len(app.txSizes) == statsSampleSize {
		app.txSizes = app.txSizes[:0]
	}
}
