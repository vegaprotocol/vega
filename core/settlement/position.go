// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

type pos struct {
	events.MarketPosition
	price *num.Uint
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

func newPos(marketPosition events.MarketPosition, price *num.Uint) *pos {
	return &pos{
		MarketPosition: marketPosition,
		price:          price.Clone(),
	}
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

func (npos) BuySumProduct() *num.Uint {
	return num.UintZero()
}

func (npos) SellSumProduct() *num.Uint {
	return num.UintZero()
}

func (npos) VWBuy() *num.Uint {
	return num.UintZero()
}

func (npos) VWSell() *num.Uint {
	return num.UintZero()
}

func (npos) AverageEntryPrice() *num.Uint {
	return num.UintZero()
}
