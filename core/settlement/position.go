// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package settlement

import (
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/pkg/errors"
)

var ErrPartyDoesNotMatch = errors.New("event party and position party do not match")

// MarketPosition stub event for network position (used in MTM stuff).
type npos struct {
	price *num.Uint
}

// See positions.MarketPosition.
type pos struct {
	// embed the type, we will copy the three main fields because those should be immutable
	// which we can't guarantee through an embedded interface
	events.MarketPosition
	party string
	size  int64
	price *num.Uint
}

// snapWrap wraps MarketPosition to implement the MarketPosition event interface so we can restore snapshots more easily.
type snapWrap struct {
	*types.MarketPosition
}

type mtmTransfer struct {
	events.MarketPosition
	transfer *types.Transfer
}

type settlementTrade struct {
	size        int64
	price       *num.Uint
	marketPrice *num.Uint
	newSize     int64 // track this so we can determine when a party switches between long <> short
}

func (t settlementTrade) Size() int64 {
	return t.size
}

func (t settlementTrade) Price() *num.Uint {
	return t.price.Clone()
}

func (t settlementTrade) MarketPrice() *num.Uint {
	return t.marketPrice.Clone()
}

func newPos(evt events.MarketPosition) *pos {
	return &pos{
		MarketPosition: evt,
		party:          evt.Party(),
		size:           evt.Size(),
		price:          evt.Price().Clone(),
	}
}

// update - set the size/price of an event accordingly.
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

func (p pos) IsEmpty() bool {
	if p.size != 0 || p.Buy() != 0 || p.Sell() != 0 {
		return false
	}
	if !p.price.IsZero() || !p.VWBuy().IsZero() || !p.VWSell().IsZero() {
		return false
	}
	return true
}

// Party - part of the MarketPosition interface, used to update position after SettlePreTrade.
func (p pos) Party() string {
	return p.party
}

// Size - part of the MarketPosition interface, used to update position after SettlePreTrade.
func (p pos) Size() int64 {
	return p.size
}

// Price - part of the MarketPosition interface, used to update position after SettlePreTrade.
func (p pos) Price() *num.Uint {
	return p.price.Clone()
}

// Transfer - part of the Transfer interface.
func (m mtmTransfer) Transfer() *types.Transfer {
	if m.transfer == nil {
		return nil
	}
	return m.transfer
}

func (npos) Party() string {
	return types.NetworkParty
}

func (npos) Size() int64 {
	return 0
}

func (npos) Buy() int64 {
	return 0
}

func (npos) Sell() int64 {
	return 0
}

func (n npos) Price() *num.Uint {
	return n.price.Clone()
}

func (npos) VWBuy() *num.Uint {
	return num.UintZero()
}

func (npos) VWSell() *num.Uint {
	return num.UintZero()
}

func (p snapWrap) Party() string {
	return p.PartyID
}

func (p snapWrap) Size() int64 {
	return p.MarketPosition.Size
}

func (p snapWrap) Buy() int64 {
	return p.MarketPosition.Buy
}

func (p snapWrap) Sell() int64 {
	return p.MarketPosition.Sell
}

func (p snapWrap) Price() *num.Uint {
	if p.MarketPosition.Price != nil {
		return p.MarketPosition.Price.Clone()
	}
	return num.UintZero()
}

func (p snapWrap) VWBuy() *num.Uint {
	if p.VwBuy != nil {
		return p.VwBuy.Clone()
	}
	return num.UintZero()
}

func (p snapWrap) VWSell() *num.Uint {
	if p.VwSell != nil {
		return p.VwSell.Clone()
	}
	return num.UintZero()
}
