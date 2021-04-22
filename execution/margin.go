package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
)

func (m *Market) calcMarginsLiquidityProvisionAmendContinuous(
	ctx context.Context, pos *positions.MarketPosition,
) error {
	asset, _ := m.mkt.GetAsset()
	market := m.GetID()

	// first we build the margin events from the collateral.
	e, err := m.collateral.GetPartyMargin(pos, asset, market)
	if err != nil {
		return err
	}

	_, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, e, m.markPrice)
	if err != nil {
		return err
	}

	// if evt is different to nil,
	// this means a margin shortfall would happen
	// we need to return an error
	if evt != nil {
		return fmt.Errorf(
			"margin would be below maintenance with amend during continuous: %w",
			ErrMarginCheckInsufficient,
		)
	}

	// any other case is fine
	return nil
}

func (m *Market) calcMarginsLiquidityProvisionAmendAuction(
	ctx context.Context, pos *positions.MarketPosition, price uint64,
) (events.Risk, error) {
	asset, _ := m.mkt.GetAsset()
	market := m.GetID()

	// first we build the margin events from the collateral.
	e, err := m.collateral.GetPartyMargin(pos, asset, market)
	if err != nil {
		return nil, err
	}

	// then we calculated margins for this party
	risk, closed, err := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, price)
	if err != nil {
		return nil, err
	}

	if len(closed) > 0 {
		// this order would take party below maintenance -> stop here
		return nil, fmt.Errorf(
			"margin would be below maintenance: %w", ErrMarginCheckInsufficient)
	}

	// in this case, if no risk event is emitted, this means
	// that the margins is covered, nothing needsto happen
	if len(risk) <= 0 {
		return nil, nil
	}

	// then we check if the required top-up is greated that the amound in
	// the GeneralBalance, if yes it means we would have to use the bond
	// account which is not acceptable at this point, we return an error as well
	if risk[0].Amount() > (risk[0].GeneralBalance() - risk[0].BondBalance()) {
		return nil, fmt.Errorf("margin would require bond: %w", ErrMarginCheckInsufficient)
	}

	return risk[0], nil
}

func (m *Market) calcMargins(ctx context.Context, pos *positions.MarketPosition, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	if m.as.InAuction() {
		return m.marginsAuction(ctx, order)
	}
	return m.margins(ctx, pos, order)
}

func (m *Market) marginsAuction(ctx context.Context, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	// 1. Get the price
	price := m.getMarkPrice(order)
	// m.log.Infof("calculating margins at %d for order at price %d", price, order.Price)
	// 2. Get all positions - we have to update margins for all traders on the book so nobody can get distressed when we eventually do uncross
	allPos := m.position.Positions()
	// 3. get the asset and ID for this market
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	// 3-b. Get position for the trader placing this order, if exists
	if cPos, ok := m.position.GetPositionByPartyID(order.PartyId); ok {
		e, err := m.collateral.GetPartyMargin(cPos, asset, mID)
		if err != nil {
			return nil, nil, err
		}
		_, closed, err := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, price)
		if err != nil {
			return nil, nil, err
		}
		if len(closed) > 0 {
			// this order would take party below maintenance -> stop here
			return nil, nil, ErrMarginCheckInsufficient
		}
		// we could transfer the funds for this trader here, but we're handling all positions lower down, including this one
		// this is just to stop all margins being updated based on a price that the trader can't even manage
	}
	// 4. construct the events for all positions + margin balances
	// at this point, we have established the order is going through
	posEvts := make([]events.Margin, 0, len(allPos))
	for _, p := range allPos {
		e, err := m.collateral.GetPartyMargin(p, asset, mID)
		if err != nil {
			// this shouldn't happen
			return nil, nil, err
		}
		posEvts = append(posEvts, e)
	}
	// 5. Get all the risk events
	risk, closed, err := m.risk.UpdateMarginAuction(ctx, posEvts, price)
	if err != nil {
		// @TODO handle this properly
		return nil, nil, err
	}
	distressed := make(map[string]struct{}, len(closed))
	mposEvts := make([]events.MarketPosition, 0, len(closed))
	for _, e := range closed {
		distressed[e.Party()] = struct{}{}
		mposEvts = append(mposEvts, e)
	}
	// 6. Attempt margin updates where possible. If position is to be closed, append it to the closed slice we already have
	riskTransfers := make([]events.Risk, 0, len(risk))
	for _, ru := range risk {
		// skip the traders with a shortfall/distressed
		if _, ok := distressed[ru.Party()]; ok {
			continue
		}
		riskTransfers = append(riskTransfers, ru)
	}
	return riskTransfers, mposEvts, nil
}

func (m *Market) margins(ctx context.Context, mpos *positions.MarketPosition, order *types.Order) ([]events.Risk, []events.MarketPosition, error) {
	price := m.getMarkPrice(order)
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, asset, mID)
	if err != nil {
		return nil, nil, err
	}
	risk, evt, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price)
	if err != nil {
		return nil, nil, err
	}
	if risk == nil {
		return nil, nil, nil
	}
	if evt != nil {
		if m.liquidity.IsPending(order.PartyId) {
			return nil, nil, ErrBondSlashing
		}
		return []events.Risk{risk}, []events.MarketPosition{evt}, nil
	}
	return []events.Risk{risk}, nil, nil
}

func (m *Market) getMarkPrice(o *types.Order) uint64 {
	// during opening auction we don't have a prior mark price, so we use the indicative price instead
	if m.as.IsOpeningAuction() {
		// we have no last known mark price
		if ip := m.matching.GetIndicativePrice(); ip != 0 {
			return ip
		}
		// we don't have an indicative price yet, this must be the first order, so we use its price
		return o.Price
	}
	return m.markPrice
}
