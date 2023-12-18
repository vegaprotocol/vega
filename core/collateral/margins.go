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

package collateral

import (
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type marginUpdate struct {
	events.MarketPosition
	margin          *types.Account
	orderMargin     *types.Account
	general         *types.Account
	lock            *types.Account
	bond            *types.Account
	asset           string
	marketID        string
	marginShortFall *num.Uint
}

func (n marginUpdate) Transfer() *types.Transfer {
	return nil
}

func (n marginUpdate) Asset() string {
	return n.asset
}

func (n marginUpdate) MarketID() string {
	return n.marketID
}

func (n marginUpdate) MarginBalance() *num.Uint {
	if n.margin == nil {
		return num.UintZero()
	}
	return n.margin.Balance.Clone()
}

func (n marginUpdate) OrderMarginBalance() *num.Uint {
	if n.orderMargin == nil {
		return num.UintZero()
	}
	return n.orderMargin.Balance.Clone()
}

// GeneralBalance here we cumulate both the general
// account and bon account so other package do not have
// to worry about how much funds are available in both
// if a bond account exists
// TODO(): maybe rename this method into AvailableBalance
// at some point if it makes senses overall the codebase.
func (n marginUpdate) GeneralBalance() *num.Uint {
	gen, bond := num.UintZero(), num.UintZero()
	if n.general != nil && n.general.Balance != nil {
		gen = n.general.Balance
	}
	if n.bond != nil && n.bond.Balance != nil {
		bond = n.bond.Balance
	}
	return num.Sum(bond, gen)
}

func (n marginUpdate) GeneralAccountBalance() *num.Uint {
	if n.general != nil && n.general.Balance != nil {
		return n.general.Balance
	}
	return num.UintZero()
}

func (n marginUpdate) MarginShortFall() *num.Uint {
	return n.marginShortFall.Clone()
}

// BondBalance - returns nil if no bond account is present, *num.Uint otherwise.
func (n marginUpdate) BondBalance() *num.Uint {
	if n.bond == nil {
		return num.UintZero()
	}
	return n.bond.Balance.Clone()
}
