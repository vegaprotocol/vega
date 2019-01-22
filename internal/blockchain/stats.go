package blockchain

type Stats struct {
	height uint64
	averageTxSize int
	//averageTxPerBatch int       // Todo(cdm): calc average TX in batch in abci.go
	totalTxLastBatch int
	totalOrdersLastBatch int
	totalTradesLastBatch int
	averageOrdersPerBatch int
	//ordersPerSecond int
	//tradesPerSecond int
}

func NewStats() *Stats {
	return &Stats{
		height: 0,
		averageTxSize: -1,
		//averageTxPerBatch: -1,    // Todo(cdm): calc average TX in batch in abci.go
		totalTxLastBatch: -1,
		totalOrdersLastBatch: -1,
		totalTradesLastBatch: -1,
		averageOrdersPerBatch: -1,
	}
}
