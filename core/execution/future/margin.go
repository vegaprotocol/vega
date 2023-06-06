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
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) calcMarginsLiquidityProvisionAmendContinuous(
	ctx context.Context, pos *positions.MarketPosition,
) error {
	market := m.GetID()

	// first we build the margin events from the collateral.
	e, err := m.collateral.GetPartyMargin(pos, m.settlementAsset, market)
	if err != nil {
		return err
	}

	_, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, e, m.getCurrentMarkPrice())
	if err != nil {
		return err
	}

	// if evt is different to nil,
	// this means a margin shortfall would happen
	// we need to return an error
	if evt != nil {
		return fmt.Errorf(
			"margin would be below maintenance with amend during continuous: %w",
			common.ErrMarginCheckInsufficient,
		)
	}

	// any other case is fine
	return nil
}

func (m *Market) calcMarginsLiquidityProvisionAmendAuction(
	ctx context.Context, pos *positions.MarketPosition, price *num.Uint,
) (events.Risk, error) {
	market := m.GetID()

	// first we build the margin events from the collateral.
	e, err := m.collateral.GetPartyMargin(pos, m.settlementAsset, market)
	if err != nil {
		return nil, err
	}

	// then we calculated margins for this party
	risk, closed := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, price)
	if len(closed) > 0 {
		// this order would take party below maintenance -> stop here
		return nil, fmt.Errorf(
			"margin would be below maintenance: %w", common.ErrMarginCheckInsufficient)
	}

	// in this case, if no risk event is emitted, this means
	// that the margins is covered, nothing needsto happen
	if len(risk) <= 0 {
		return nil, nil
	}

	// then we check if the required top-up is greated that the amound in
	// the GeneralBalance, if yes it means we would have to use the bond
	// account which is not acceptable at this point, we return an error as well
	if risk[0].Amount().GT(num.UintZero().Sub(risk[0].GeneralBalance(), risk[0].BondBalance())) {
		return nil, fmt.Errorf("margin would require bond: %w", common.ErrMarginCheckInsufficient)
	}

	return risk[0], nil
}

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
	return m.risk.UpdateMarginsOnSettlement(ctx, margins, price)
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
	risk, closed := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, m.getMarketObservable(order.Price.Clone()))
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
	risk, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price.Clone())
	if err != nil {
		return nil, nil, err
	}
	if risk == nil {
		return nil, nil, nil
	}
	if evt != nil {
		if m.liquidity.IsPending(order.Party) {
			return nil, nil, ErrBondSlashing
		}
		return []events.Risk{risk}, []events.MarketPosition{evt}, nil
	}
	return []events.Risk{risk}, nil, nil
}
