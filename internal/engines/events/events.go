// this package just centralises all interfaces that the various
// engines will use to "talk" to eachother. More often than not, these
// interfaces will end up embedding the previous ones
package events

// MarketPosition - The main position interface
type MarketPosition interface {
	Party() string
	Size() int64
	Price() uint64
}

// MarginChange - change made to balances after Settling MTM
type MarginChange interface {
	MarketPosition
	Asset() string
	MarginBalance() uint64
	GeneralBalance() uint64
}

// RiskUpdate summarizes everything + an eventual update to margin account
type RiskUpdate interface {
	MarginChange
	Update() int64
}
