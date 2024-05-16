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

package future

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func (m *Market) updateIsolatedMarginsOnPositionChange(ctx context.Context, mpos *positions.MarketPosition, order *types.Order, trade *types.Trade) error {
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, m.GetID())
	if err != nil {
		return err
	}
	price := m.getMarketObservable(order.Price.Clone())
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	orders := m.matching.GetOrdersPerParty(order.Party)
	marginFactor := m.getMarginFactor(order.Party)
	r, err := m.risk.UpdateIsolatedMarginsOnPositionChange(ctx, pos, price, increment, orders, []*types.Trade{trade}, order.Side, marginFactor)
	if err != nil {
		return err
	}
	for _, rr := range r {
		m.transferMargins(ctx, []events.Risk{rr}, nil)
	}
	pos, err = m.collateral.GetPartyMargin(mpos, m.settlementAsset, m.GetID())
	if err != nil {
		return err
	}
	_, err = m.risk.CheckMarginInvariants(ctx, pos, price, increment, orders, marginFactor)
	return err
}

func (m *Market) getIsolatedMarginContext(mpos *positions.MarketPosition, order *types.Order) (*num.Uint, events.Margin, num.Decimal, *num.Uint, num.Decimal, []*types.Order, error) {
	var orderPrice *num.Uint
	if order != nil {
		orderPrice = order.Price.Clone()
	} else {
		orderPrice = num.UintZero()
	}
	marketObservable := m.getMarketObservable(orderPrice)
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, mID)
	if err != nil {
		return nil, nil, num.DecimalZero(), nil, num.DecimalZero(), nil, err
	}
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	auctionPrice := m.getAuctionPrice()
	marginFactor := m.getMarginFactor(mpos.Party())
	orders := m.matching.GetOrdersPerParty(mpos.Party())
	return marketObservable, pos, increment, auctionPrice, marginFactor, orders, nil
}

func (m *Market) getAuctionPrice() *num.Uint {
	var auctionPrice *num.Uint
	if m.as.InAuction() {
		if m.capMax != nil && m.fCap.FullyCollateralised {
			// if this is a capped market with max price, this is the price we need to use all the time
			// this function is called to calculate margins, and margin calculations are always going to be based on the max price.
			return m.capMax.Clone()
		}
		auctionPrice = m.matching.GetIndicativePrice()
		if markPrice := m.getCurrentMarkPrice(); markPrice != nil && !markPrice.IsZero() && (markPrice.GT(auctionPrice) || auctionPrice == nil) {
			auctionPrice = markPrice
		}
	}
	return auctionPrice
}
