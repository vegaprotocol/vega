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

func (m *Market) updateIsolatedMarginsOnPositionChange(ctx context.Context, mpos *positions.MarketPosition, order *types.Order, trade *types.Trade) (events.Risk, error) {
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, m.GetID())
	if err != nil {
		return nil, err
	}
	price := m.getMarketObservable(order.Price.Clone())
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	return m.risk.UpdateIsolatedMarginsOnPositionChange(ctx, pos, price, increment, m.matching.GetOrdersPerParty(order.Party), []*types.Trade{trade}, order.Side, m.getMarginFactor(order.Party), nil)
}

func (m *Market) calcIsolatedMarginsOnNewOrderTraded(ctx context.Context, mpos *positions.MarketPosition, newOrder *types.Order, trades []*types.Trade) ([]events.Risk, error) {
	price := m.getMarketObservable(newOrder.Price.Clone())
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, mID)
	if err != nil {
		return nil, err
	}
	marginFactor := m.getMarginFactor(mpos.Party())
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	orders := m.matching.GetOrdersPerParty(mpos.Party())
	risk, err := m.risk.CheckIsolatedMarginsOnNewOrderTraded(ctx, pos, price, nil, increment, orders, trades, marginFactor, m.positionFactor, newOrder.Side)
	if err != nil {
		return nil, err
	}
	if risk == nil {
		return nil, nil
	}
	return []events.Risk{risk}, nil
}

func (m *Market) calcIsolatedMarginsOrder(ctx context.Context, mpos *positions.MarketPosition, newOrder *types.Order, orders []*types.Order) ([]events.Risk, error) {
	price := m.getMarketObservable(newOrder.Price.Clone())
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, mID)
	if err != nil {
		return nil, err
	}
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())

	auctionPrice := m.matching.GetIndicativePrice()
	if markPrice := m.getCurrentMarkPrice(); markPrice != nil && !markPrice.IsZero() && markPrice.GT(auctionPrice) {
		auctionPrice = markPrice
	}

	marginFactor := m.getMarginFactor(mpos.Party())
	risk, err := m.risk.CheckIsolatedMargins(ctx, pos, orders, price, auctionPrice, increment, marginFactor, m.positionFactor)
	if err != nil {
		return nil, err
	}
	if risk == nil {
		return nil, nil
	}
	return []events.Risk{risk}, nil
}

func (m *Market) recalcIsolatedMargins(ctx context.Context, mpos *positions.MarketPosition, orders []*types.Order) ([]events.Risk, error) {
	price := m.getMarketObservable(num.UintZero())
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, mID)
	if err != nil {
		return nil, err
	}
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())

	auctionPrice := m.matching.GetIndicativePrice()
	if markPrice := m.getCurrentMarkPrice(); markPrice != nil && !markPrice.IsZero() && markPrice.GT(auctionPrice) {
		auctionPrice = markPrice
	}

	marginFactor := m.getMarginFactor(mpos.Party())
	risk, err := m.risk.CheckIsolatedMargins(ctx, pos, orders, price, auctionPrice, increment, marginFactor, m.positionFactor)
	if err != nil {
		return nil, err
	}
	if risk == nil {
		return nil, nil
	}
	return []events.Risk{risk}, nil
}
