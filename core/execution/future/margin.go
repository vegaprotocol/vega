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
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) calcMargins(ctx context.Context, pos *positions.MarketPosition, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	if m.as.InAuction() {
		return m.marginsAuction(ctx, order)
	}
	return m.margins(ctx, pos, order)
}

func (m *Market) updateMargin(ctx context.Context, pos []events.MarketPosition) []events.Risk {
	price := m.getCurrentMarkPrice()
	mID := m.GetID()
	margins := make([]events.Margin, 0, len(pos))
	for _, p := range pos {
		e, err := m.collateral.GetPartyMargin(p, m.settlementAsset, mID)
		if err != nil {
			m.log.Error("Failed to get margin event for party position",
				logging.String("party", p.Party()),
				logging.Error(err),
			)
			continue
		}
		// add the required margin event
		margins = append(margins, e)
	}
	// we should get any and all risk events we need here
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	return m.risk.UpdateMarginsOnSettlement(ctx, margins, price, increment, m.getAuctionPrice())
}

func (m *Market) marginsAuction(ctx context.Context, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	cPos, ok := m.position.GetPositionByPartyID(order.Party)
	if !ok {
		return nil, nil, nil
	}
	mID := m.GetID()
	e, err := m.collateral.GetPartyMargin(cPos, m.settlementAsset, mID)
	if err != nil {
		return nil, nil, err
	}
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	risk, closed := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, m.getMarketObservable(order.Price.Clone()), increment, m.getAuctionPrice())
	if len(closed) > 0 {
		// this order would take party below maintenance -> stop here
		return nil, nil, common.ErrMarginCheckInsufficient
	}
	return risk, nil, nil
}

func (m *Market) margins(ctx context.Context, mpos *positions.MarketPosition, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	price := m.getMarketObservable(order.Price.Clone())
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, m.settlementAsset, mID)
	if err != nil {
		return nil, nil, err
	}
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	risk, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price.Clone(), increment, m.getAuctionPrice())
	if err != nil {
		return nil, nil, err
	}
	if risk == nil {
		return nil, nil, nil
	}
	if evt != nil {
		return []events.Risk{risk}, []events.MarketPosition{evt}, nil
	}
	return []events.Risk{risk}, nil, nil
}
