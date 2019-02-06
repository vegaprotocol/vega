package blockchain

type Stats struct {
	height uint64
	averageTxSizeBytes int
	averageTxPerBatch int
	totalTxLastBatch int
	totalOrdersLastBatch int
	totalTradesLastBatch int
	averageOrdersPerBatch int
	//ordersPerSecond int        // --
	//tradesPerSecond int        // todo(cdm): requires timing, devoid from blocks
}

func NewStats() *Stats {
	return &Stats{
		height: 0,
		averageTxSizeBytes: 0,
		averageTxPerBatch: 0,
		totalTxLastBatch: 0,
		totalOrdersLastBatch: 0,
		totalTradesLastBatch: 0,
		averageOrdersPerBatch: 0,
	}
}

func (s *Stats) Height() uint64 {
	return s.height
}

func (s *Stats) AverageTxSizeBytes() int {
	return s.averageTxSizeBytes
}

func (s *Stats) AverageTxPerBatch() int {
	return s.averageTxPerBatch
}

func (s *Stats) TotalTxLastBatch() int {
	return s.totalTxLastBatch
}

func (s *Stats) TotalOrdersLastBatch() int {
	return s.totalOrdersLastBatch
}

func (s *Stats) TotalTradesLastBatch() int {
	return s.totalTradesLastBatch
}

func (s *Stats) AverageOrdersPerBatch() int {
	return s.averageOrdersPerBatch
}