package blockchain

import "sync/atomic"

// Stats hold stats over all the vega node
type Stats struct {
	height                uint64
	averageTxSizeBytes    int
	averageTxPerBatch     int
	totalTxCurrentBatch   int
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
	blockDuration         uint64 // nanoseconds
}

// NewStats instantiate a new Stats
func NewStats() *Stats {
	return &Stats{
		height:                0,
		averageTxSizeBytes:    0,
		averageTxPerBatch:     0,
		totalTxCurrentBatch:   0,
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

// Height returns the current heights of the chain
func (s *Stats) Height() uint64 {
	return s.height
}

// IncHeight increment the height of the chain
func (s *Stats) IncHeight() {
	s.height++
}

// AverageTxSizeBytes return the average size in bytes of the
// transaction sent to vega
func (s *Stats) AverageTxSizeBytes() int {
	return s.averageTxSizeBytes
}

func (s *Stats) SetAverageTxSizeBytes(i int) {
	s.averageTxSizeBytes = i
}

// AverageTxPerBatch return the average number of
// transaction per block
func (s *Stats) AverageTxPerBatch() int {
	return s.averageTxPerBatch
}

func (s *Stats) SetAverageTxPerBatch(i int) {
	s.averageTxPerBatch = i
}

// TotalTxLastBatch return the number of transaction
// processed in the last accepted block in the chain
func (s *Stats) TotalTxLastBatch() int {
	return s.totalTxLastBatch
}

func (s *Stats) SetTotalTxLastBatch(i int) {
	s.totalTxLastBatch = i
}

func (s *Stats) SetTotalTxCurrentBatch(i int) {
	s.totalTxCurrentBatch = i
}

func (s *Stats) TotalTxCurrentBatch() int {
	return s.totalTxCurrentBatch
}

func (s *Stats) IncTotalTxCurrentBatch() {
	s.totalTxCurrentBatch++
}

// TotalOrdersLastBatch returns the number of orders
// accepted in the last block in the chain
func (s *Stats) TotalOrdersLastBatch() int {
	return s.totalOrdersLastBatch
}

// TotalTradesLastBatch returns the number of trades
// created during the last block in the chain
func (s *Stats) TotalTradesLastBatch() int {
	return s.totalTradesLastBatch
}

// AverageOrdersPerBatch returns the average number
// of orders accepted per blocks
func (s *Stats) AverageOrdersPerBatch() int {
	return s.averageOrdersPerBatch
}

// TotalAmendOrder returns the total amount of order
// amended processed by the vega node
func (s *Stats) TotalAmendOrder() uint64 {
	return atomic.LoadUint64(&s.totalAmendOrder)
}

// TotalCancelOrder return the total number of orders
// cancel by the vega node
func (s *Stats) TotalCancelOrder() uint64 {
	return atomic.LoadUint64(&s.totalCancelOrder)
}

// TotalCreateOrder return the total amount of
// request to create a new order
func (s *Stats) TotalCreateOrder() uint64 {
	return atomic.LoadUint64(&s.totalCreateOrder)
}

// TotalOrders return the total amount of
// orders placed in the system
func (s *Stats) TotalOrders() uint64 {
	return atomic.LoadUint64(&s.totalOrders)
}

// TotalTrades return the total amount of trades
// in the system
func (s *Stats) TotalTrades() uint64 {
	return atomic.LoadUint64(&s.totalTrades)
}

// OrdersPerSecond return the total number of orders
// processed during the last second
func (s *Stats) OrdersPerSecond() uint64 {
	return atomic.LoadUint64(&s.ordersPerSecond)
}

// TradesPerSecond return the total number of trades
// generated during the last second
func (s *Stats) TradesPerSecond() uint64 {
	return atomic.LoadUint64(&s.tradesPerSecond)
}

// BlockDuration return the duration it took
// to generate the last block
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
