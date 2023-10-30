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
	var currentOrders, currentTrades, eps uint64
	app.stats.SetBlockDuration(uint64(duration * float64(time.Second.Nanoseconds())))
	if duration > 0 {
		currentOrders, currentTrades, eps = uint64(float64(app.stats.CurrentOrdersInBatch())/duration),
			uint64(float64(app.stats.CurrentTradesInBatch())/duration), uint64(float64(app.stats.CurrentEventsInBatch())/duration)
	}
	app.stats.SetOrdersPerSecond(currentOrders)
	app.stats.SetTradesPerSecond(currentTrades)
	app.stats.SetEventsPerSecond(eps)
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
		logging.Uint64("events-per-sec", eps),
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
