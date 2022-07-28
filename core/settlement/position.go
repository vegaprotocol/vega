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
	"code.vegaprotocol.io/vega/core/types/num"

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
	party   string
	size    int64
	price   *num.Uint
	newSize int64 // track this so we can determine when a party switches between long <> short
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
	return num.Zero()
}

func (npos) VWSell() *num.Uint {
	return num.Zero()
}
