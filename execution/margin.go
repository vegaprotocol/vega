package execution

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
)

func (m *Market) calcMargins(ctx context.Context, pos *positions.MarketPosition, order *types.Order) (*types.Transfer, error) {
	if m.as.InAuction() {
		return m.marginsAuction(ctx, order)
	}
	return m.margins(ctx, pos, order)
}

func (m *Market) marginsAuction(ctx context.Context, order *types.Order) (*types.Transfer, error) {
	// 1. Get the price
	price := m.getMarkPrice(order)
	m.log.Infof("calculating margins at %d for order at price %d", price, order.Price)
	// 2. Get all positions - we have to update margins for all traders on the book so nobody can get distressed when we eventually do uncross
	allPos := m.position.Positions()
	// 3. get the asset and ID for this market
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	// 3-b. Get position for the trader placing this order, if exists
	if cPos, ok := m.position.GetPositionByPartyID(order.PartyId); ok {
		e, err := m.collateral.GetPartyMargin(cPos, asset, mID)
		if err != nil {
			return nil, err
		}
		_, closed, err := m.risk.UpdateMarginAuction(ctx, []events.Margin{e}, price)
		if err != nil {
			return nil, err
		}
		if len(closed) > 0 {
			// this order would take party below maintenance -> stop here
			return nil, ErrMarginCheckInsufficient
		}
		// we could transfer the funds for this trader here, but we're handling all positions lower down, including this one
		// this is just to stop all margins being updated based on a price that the trader can't even manage
	}
	// 4. construct the events for all positions + margin balances
	posEvts := make([]events.Margin, 0, len(allPos))
	for _, p := range allPos {
		e, err := m.collateral.GetPartyMargin(p, asset, mID)
		if err != nil {
			// this shouldn't happen
			return nil, err
		}
		posEvts = append(posEvts, e)
	}
	// 5. Get all the risk events
	risk, closed, err := m.risk.UpdateMarginAuction(ctx, posEvts, price)
	if err != nil {
		// @TODO handle this properly
		return nil, err
	}
	mposEvts := make([]events.MarketPosition, 0, len(closed))
	for _, e := range closed {
		mposEvts = append(mposEvts, e)
	}
	// 6. Attempt margin updates where possible. If position is to be closed, append it to the closed slice we already have
	marginEvts := make([]events.Event, 0, len(risk))
	for _, ru := range risk {
		tr, _, err := m.collateral.MarginUpdateOnOrder(ctx, mID, ru)
		if err != nil {
			// @TODO handle this
			return nil, err
		}
		marginEvts = append(marginEvts, events.NewTransferResponse(ctx, []*types.TransferResponse{tr}))
	}
	// 7. Send batch of Transfer events out
	m.broker.SendBatch(marginEvts)
	// 8. Close out untenable positions
	rmorders, err := m.matching.RemoveDistressedOrders(mposEvts)
	if err != nil {
		return nil, err
	}
	evts := make([]events.Event, 0, len(rmorders))
	for _, o := range rmorders {
		// cancel order
		o.Status = types.Order_STATUS_CANCELLED
		// create event
		evts = append(evts, events.NewOrderEvent(ctx, o))
		// remove order from positions
		m.position.UnregisterOrder(o)
	}
	m.broker.SendBatch(evts)
	return nil, nil
}

func (m *Market) margins(ctx context.Context, mpos *positions.MarketPosition, order *types.Order) (*types.Transfer, error) {
	price := m.getMarkPrice(order)
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, asset, mID)
	if err != nil {
		return nil, err
	}
	risk, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price)
	if err != nil {
		return nil, err
	}
	if risk == nil {
		return nil, nil
	}
	tr, closed, err := m.collateral.MarginUpdateOnOrder(ctx, mID, risk)
	if err != nil {
		return nil, err
	}
	if closed != nil && closed.MarginShortFall() > 0 {
		// @TODO handle closed
		return nil, nil
	}
	if tr == nil {
		return nil, nil
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tr}))
	// create the rollback transaction
	// for some reason, we can get a transfer object returned, but no actual transfers?
	var riskRollback *types.Transfer
	if len(tr.Transfers) > 0 {
		riskRollback = &types.Transfer{
			Owner: risk.Party(),
			Amount: &types.FinancialAmount{
				Amount: int64(tr.Transfers[0].Amount),
				Asset:  asset,
			},
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_HIGH,
			MinAmount: int64(tr.Transfers[0].Amount),
		}
	}
	return riskRollback, nil
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
