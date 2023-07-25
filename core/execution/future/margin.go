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
	return m.risk.UpdateMarginsOnSettlement(ctx, margins, price, increment)
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
	risk, closed := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, m.getMarketObservable(order.Price.Clone()), increment)
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
	risk, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price.Clone(), increment)
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
