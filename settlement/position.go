package settlement

import (
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/pkg/errors"
)

var (
	ErrPartyDoesNotMatch = errors.New("event party and position party do not match")
)

// See positions.MarketPosition
type pos struct {
	// embed the type, we will copy the three main fields because those should be immutable
	// which we can't guarantee through an embedded interface
	events.MarketPosition
	party   string
	size    int64
	price   *num.Uint
	newSize int64 // track this so we can determine when a trader switches between long <> short
}

type mtmTransfer struct {
	events.MarketPosition
	transfer *types.Transfer
}

func newPos(evt events.MarketPosition) *pos {
	return &pos{
		MarketPosition: evt,
		party:          evt.Party(),
		size:           evt.Size(),
		price:          evt.Price().Clone(),
	}
}

// update - set the size/price of an event accordingly
func (p *pos) update(evt events.MarketPosition) error {
	// this check, in theory, should not be needed...
	if p.party != evt.Party() {
		return ErrPartyDoesNotMatch
	}
	// embed updated event
	p.MarketPosition = evt
	p.size = evt.Size()
	p.price = evt.Price().Clone()
	return nil
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
func (p pos) Price() *num.Uint {
	return p.price.Clone()
}

// Transfer - part of the Transfer interface
func (m mtmTransfer) Transfer() *types.Transfer {
	if m.transfer == nil {
		return nil
	}
	return m.transfer
}
