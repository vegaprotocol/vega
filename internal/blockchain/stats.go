package blockchain

import "sync/atomic"

type Stats struct {
	height                uint64
	averageTxSizeBytes    int
	averageTxPerBatch     int
	totalTxLastBatch      int
	totalOrdersLastBatch  int
	totalTradesLastBatch  int
	averageOrdersPerBatch int
	ordersPerSecond       uint64
	tradesPerSecond       uint64
	totalAmendOrder       uint64
	totalCancelOrder      uint64
	totalCreateOrder      uint64
	totalOrders           uint64
	totalTrades           uint64
	blockDuration         uint64
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
		blockDuration:         0,
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
	return atomic.LoadUint64(&s.totalAmendOrder)
}

func (s *Stats) TotalCancelOrder() uint64 {
	return atomic.LoadUint64(&s.totalCancelOrder)
}

func (s *Stats) TotalCreateOrder() uint64 {
	return atomic.LoadUint64(&s.totalCreateOrder)
}

func (s *Stats) TotalOrders() uint64 {
	return atomic.LoadUint64(&s.totalOrders)
}

func (s *Stats) TotalTrades() uint64 {
	return atomic.LoadUint64(&s.totalTrades)
}

func (s *Stats) OrdersPerSecond() uint64 {
	return atomic.LoadUint64(&s.ordersPerSecond)
}

func (s *Stats) TradesPerSecond() uint64 {
	return atomic.LoadUint64(&s.tradesPerSecond)
}

func (s *Stats) BlockDuration() uint64 {
	return atomic.LoadUint64(&s.blockDuration)
}

func (s *Stats) addTotalAmendOrder(val uint64) uint64 {
	return atomic.AddUint64(&s.totalAmendOrder, val)
}

func (s *Stats) addTotalCancelOrder(val uint64) uint64 {
	return atomic.AddUint64(&s.totalCancelOrder, val)
}

func (s *Stats) addTotalCreateOrder(val uint64) uint64 {
	return atomic.AddUint64(&s.totalCreateOrder, val)
}

func (s *Stats) addTotalOrders(val uint64) uint64 {
	return atomic.AddUint64(&s.totalOrders, val)
}

func (s *Stats) addTotalTrades(val uint64) uint64 {
	return atomic.AddUint64(&s.totalTrades, val)
}

func (s *Stats) setOrdersPerSecond(val uint64) {
	atomic.StoreUint64(&s.ordersPerSecond, val)
}

func (s *Stats) setTradesPerSecond(val uint64) {
	atomic.StoreUint64(&s.tradesPerSecond, val)
}

func (s *Stats) setBlockDuration(val uint64) {
	atomic.StoreUint64(&s.blockDuration, val)
}
