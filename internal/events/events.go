package events

import (
	types "code.vegaprotocol.io/vega/proto"
)

// MarketPosition is an event with a change to a position.
type MarketPosition interface {
	Party() string
	Size() int64
	Buy() int64
	Sell() int64
	Price() uint64
}

// Transfer is an event passed on by settlement engine, contains position
// and the resulting transfer for the collateral engine to use. We need MarketPosition
// because we can't loose the long/short status of the open positions.
type Transfer interface {
	MarketPosition
	Transfer() *types.Transfer
}

// Margin is an event with a change to balances after settling e.g. MTM.
type Margin interface {
	MarketPosition
	Asset() string
	MarginBalance() uint64
	GeneralBalance() uint64
	MarketID() string
}

// Risk is an event that summarizes everything and an eventual update to margin account.
type Risk interface {
	Margin
	Amount() int64
	Transfer() *types.Transfer // I know, it's included in the Transfer interface, but this is to make it clear that this particular func is masked at this level
}
