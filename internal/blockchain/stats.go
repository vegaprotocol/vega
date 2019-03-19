package blockchain

type Stats struct {
	height                uint64
	averageTxSizeBytes    int
	averageTxPerBatch     int
	totalTxLastBatch      int
	totalOrdersLastBatch  int
	totalTradesLastBatch  int
	averageOrdersPerBatch int
	//ordersPerSecond int        // --
	//tradesPerSecond int        // requires timing, devoid from blocks
	totalAmendOrder  uint64
	totalCancelOrder uint64
	totalCreateOrder uint64
	totalOrders      uint64
	totalTrades      uint64
}

func NewStats() *Stats {
	return &Stats{
		height:                0,
		averageTxSizeBytes:    0,
		averageTxPerBatch:     0,
		totalTxLastBatch:      0,
		totalOrdersLastBatch:  0,
		totalTradesLastBatch:  0,
		averageOrdersPerBatch: 0,
		totalAmendOrder:       0,
		totalCancelOrder:      0,
		totalCreateOrder:      0,
		totalOrders:           0,
		totalTrades:           0,
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

func (s *Stats) TotalAmendOrder() uint64 {
	return s.totalAmendOrder
}

func (s *Stats) TotalCancelOrder() uint64 {
	return s.totalCancelOrder
}

func (s *Stats) TotalCreateOrder() uint64 {
	return s.totalCreateOrder
}

func (s *Stats) TotalOrders() uint64 {
	return s.totalOrders
}

func (s *Stats) TotalTrades() uint64 {
	return s.totalTrades
}
