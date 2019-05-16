package settlement

import (
	"code.vegaprotocol.io/vega/internal/engines/events"
	types "code.vegaprotocol.io/vega/proto"
)

type pos struct {
	party string
	size  int64
	price uint64
}

type mtmTransfer struct {
	events.MarketPosition
	transfer *types.Transfer
}

// Party - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Party() string {
	return p.party
}

// Size - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Size() int64 {
	return p.size
}

// Price - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Price() uint64 {
	return p.price
}

// Transfer - part of the MTMTransfer interface
func (m mtmTransfer) Transfer() *types.Transfer {
	return m.transfer
}
