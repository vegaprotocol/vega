// this package just centralises all interfaces that the various
// engines will use to "talk" to eachother. More often than not, these
// interfaces will end up embedding the previous ones
package events

import (
	types "code.vegaprotocol.io/vega/proto"
)

// MarketPosition - The main position interface
type MarketPosition interface {
	Party() string
	Size() int64
	Price() uint64
}

// MTMTransfer, the interface passed on by settlement engine, contains position
// and the resulting transfer for the collateral engine to use. We need MarketPosition
// because we can't loose the long/short status of the open positions
type MTMTransfer interface {
	MarketPosition
	Transfer() *types.Transfer
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
	Amount() int64
}
