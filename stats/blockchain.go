package stats

// Blockchain hold stats over all the vega node
type Blockchain struct {
	height                uint64
	averageTxSizeBytes    int
	averageTxPerBatch     int
	totalTxCurrentBatch   int
	totalTxLastBatch      int
	totalOrdersLastBatch  int
	totalTradesLastBatch  int
	averageOrdersPerBatch int
	currentOrdersInBatch  uint64
	currentTradesInBatch  uint64
	totalBatches          uint64
	ordersPerSecond       uint64
	tradesPerSecond       uint64
	totalAmendOrder       uint64
	totalCancelOrder      uint64
	totalCreateOrder      uint64
	totalOrders           uint64
	totalTrades           uint64
	blockDuration         uint64 // nanoseconds
}

// NewBlockchain instantiate a new Blockchain
func NewBlockchain() *Blockchain {
	return &Blockchain{}
}

// IncTotalBatches increment total batches
func (b *Blockchain) IncTotalBatches() {
	b.totalBatches++
}

// TotalBatches get total batches
func (b Blockchain) TotalBatches() uint64 {
	return b.totalBatches
}

func (b *Blockchain) NewBatch() {
	b.totalOrdersLastBatch = int(b.currentOrdersInBatch)
	b.totalTradesLastBatch = int(b.currentTradesInBatch)
	b.currentOrdersInBatch = 0
	b.currentTradesInBatch = 0
}

func (b *Blockchain) ResetBatchTotals() {
	b.currentOrdersInBatch = 0
	b.currentTradesInBatch = 0
}

func (b *Blockchain) IncCurrentOrdersInBatch() {
	b.currentOrdersInBatch++
}

func (b *Blockchain) AddCurrentTradesInBatch(i int) {
	b.currentTradesInBatch += uint64(i)
}

func (b Blockchain) CurrentOrdersInBatch() uint64 {
	return b.currentOrdersInBatch
}

func (b Blockchain) CurrentTradesInBatch() uint64 {
	return b.currentTradesInBatch
}

// Height returns the current heights of the chain
func (b *Blockchain) Height() uint64 {
	return b.height
}

// IncHeight increment the height of the chain
func (b *Blockchain) IncHeight() {
	b.height++
}

// AverageTxSizeBytes return the average size in bytes of the
// transaction sent to vega
func (b *Blockchain) AverageTxSizeBytes() int {
	return b.averageTxSizeBytes
}

func (b *Blockchain) SetAverageTxSizeBytes(i int) {
	b.averageTxSizeBytes = i
}

// AverageTxPerBatch return the average number of
// transaction per block
func (b *Blockchain) AverageTxPerBatch() int {
	return b.averageTxPerBatch
}

func (b *Blockchain) SetAverageTxPerBatch(i int) {
	b.averageTxPerBatch = i
}

// TotalTxLastBatch return the number of transaction
// processed in the last accepted block in the chain
func (b *Blockchain) TotalTxLastBatch() int {
	return b.totalTxLastBatch
}

func (b *Blockchain) SetTotalTxLastBatch(i int) {
	b.totalTxLastBatch = i
}

func (b *Blockchain) SetTotalTxCurrentBatch(i int) {
	b.totalTxCurrentBatch = i
}

func (b *Blockchain) TotalTxCurrentBatch() int {
	return b.totalTxCurrentBatch
}

func (b *Blockchain) IncTotalTxCurrentBatch() {
	b.totalTxCurrentBatch++
}

// SetTotalOrdersLastBatch assing total orders
func (b *Blockchain) SetTotalOrdersLastBatch(i int) {
	b.totalOrdersLastBatch = i
}

// TotalOrdersLastBatch returns the number of orders
// accepted in the last block in the chain
func (b Blockchain) TotalOrdersLastBatch() int {
	return b.totalOrdersLastBatch
}

// SetTotalTradesLastBatch set total trades
func (b *Blockchain) SetTotalTradesLastBatch(i int) {
	b.totalTradesLastBatch = i
}

// TotalTradesLastBatch returns the number of trades
// created during the last block in the chain
func (b Blockchain) TotalTradesLastBatch() int {
	return b.totalTradesLastBatch
}

// SetAverageOrdersPerBatch sets new average orders per batch
func (b *Blockchain) SetAverageOrdersPerBatch(i int) {
	b.averageOrdersPerBatch = i
}

// AverageOrdersPerBatch returns the average number
// of orders accepted per blocks
func (b Blockchain) AverageOrdersPerBatch() int {
	return b.averageOrdersPerBatch
}

// TotalAmendOrder returns the total amount of order
// amended processed by the vega node
func (b Blockchain) TotalAmendOrder() uint64 {
	return b.totalAmendOrder
}

// TotalCancelOrder return the total number of orders
// cancel by the vega node
func (b Blockchain) TotalCancelOrder() uint64 {
	return b.totalCancelOrder
}

// TotalCreateOrder return the total amount of
// request to create a new order
func (b Blockchain) TotalCreateOrder() uint64 {
	return b.totalCreateOrder
}

// TotalOrders return the total amount of
// orders placed in the system
func (b Blockchain) TotalOrders() uint64 {
	return b.totalOrders
}

// TotalTrades return the total amount of trades
// in the system
func (b Blockchain) TotalTrades() uint64 {
	return b.totalTrades
}

// OrdersPerSecond return the total number of orders
// processed during the last second
func (b Blockchain) OrdersPerSecond() uint64 {
	return b.ordersPerSecond
}

// TradesPerSecond return the total number of trades
// generated during the last second
func (b Blockchain) TradesPerSecond() uint64 {
	return b.tradesPerSecond
}

// BlockDuration return the duration it took
// to generate the last block
func (b Blockchain) BlockDuration() uint64 {
	return b.blockDuration
}

func (b *Blockchain) IncTotalAmendOrder() {
	b.totalAmendOrder++
}

func (b *Blockchain) AddTotalAmendOrder(val uint64) uint64 {
	r := val + b.totalAmendOrder
	b.totalAmendOrder = r
	return r
}

func (b *Blockchain) IncTotalCancelOrder() {
	b.totalCancelOrder++
}

func (b *Blockchain) AddTotalCancelOrder(val uint64) uint64 {
	r := b.totalCancelOrder + val
	b.totalCancelOrder = r
	return r
}

func (b *Blockchain) IncTotalCreateOrder() {
	b.totalCreateOrder++
}

// AddTotalCreateOrder - increment total created orders
func (b *Blockchain) AddTotalCreateOrder(val uint64) uint64 {
	r := b.totalCreateOrder + val
	b.totalCreateOrder = r
	return r
}

func (b *Blockchain) IncTotalOrders() {
	b.totalOrders++
}

// AddTotalOrders increment total orders
func (b *Blockchain) AddTotalOrders(val uint64) uint64 {
	r := b.totalOrders + val
	b.totalOrders = r
	return r
}

// AddTotalTrades increment total trades
func (b *Blockchain) AddTotalTrades(val uint64) uint64 {
	r := b.totalTrades + val
	b.totalTrades = r
	return r
}

func (b *Blockchain) SetOrdersPerSecond(val uint64) {
	b.ordersPerSecond = val
}

func (b *Blockchain) SetTradesPerSecond(val uint64) {
	b.tradesPerSecond = val
}

func (b *Blockchain) SetBlockDuration(val uint64) {
	b.blockDuration = val
}
