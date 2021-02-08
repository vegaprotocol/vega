package execution

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
)

func (m *Market) calcMargins(ctx context.Context, pos *positions.MarketPosition, order *types.Order, failOnLPMarginShortfall bool) (error, events.Margin) {
	if m.as.InAuction() {
		return m.marginsAuction(ctx, order), nil
	}
	return m.margins(ctx, pos, order, failOnLPMarginShortfall)
}

func (m *Market) marginsAuction(ctx context.Context, order *types.Order) error {
	// 1. Get the price
	price := m.getMarkPrice(order)
	// 2. Get all positions - we have to update margins for all traders on the book so nobody can get distressed when we eventually do uncross
	allPos := m.position.Positions()
	// 3. get the asset and ID for this market
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	// 4. construct the events for all positions + margin balances
	posEvts := make([]events.Margin, 0, len(allPos))
	for _, p := range allPos {
		e, err := m.collateral.GetPartyMargin(p, asset, mID)
		if err != nil {
			// this shouldn't happen
			return err
		}
		posEvts = append(posEvts, e)
	}
	// 5. Get all the risk events
	risk, closed, err := m.risk.UpdateMarginAuction(ctx, posEvts, price)
	if err != nil {
		// @TODO handle this properly
		return err
	}
	mposEvts := make([]events.MarketPosition, 0, len(closed))
	for _, e := range closed {
		mposEvts = append(mposEvts, e)
	}
	// 6. Attempt margin updates where possible. If position is to be closed, append it to the closed slice we already have
	marginEvts := make([]events.Event, 0, len(risk))
	for _, ru := range risk {
		tr, closeP, err := m.collateral.MarginUpdateOnOrder(ctx, mID, ru)
		if err != nil {
			// @TODO handle this
			return err
		}
		if closeP != nil {
			mposEvts = append(mposEvts, closeP)
			continue
		}
		marginEvts = append(marginEvts, events.NewTransferResponse(ctx, []*types.TransferResponse{tr}))
	}
	// 7. Send batch of Transfer events out
	m.broker.SendBatch(marginEvts)
	// 8. Close out untenable positions
	rmorders, err := m.matching.RemoveDistressedOrders(mposEvts)
	if err != nil {
		return err
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
	return nil
}

func (m *Market) margins(ctx context.Context, mpos *positions.MarketPosition, order *types.Order, failOnLPMarginShortfall bool) (error, events.Margin) {
	price := m.getMarkPrice(order)
	asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	pos, err := m.collateral.GetPartyMargin(mpos, asset, mID)
	if err != nil {
		return err, nil
	}
	risk, err := m.risk.UpdateMarginOnNewOrder(ctx, pos, price)
	if err != nil {
		return err, nil
	}
	if risk == nil {
		return nil, nil
	}
	tr, bondPenalty, err := m.collateral.MarginUpdateOnOrder(ctx, mID, risk)
	if bondPenalty != nil {
		// if closePose is not nil then we return an error as well, it means the trader did not have enough
		// monies to reach the InitialMargin
		shortfall := bondPenalty.MarginShortFall()
		if shortfall > 0 {
			if failOnLPMarginShortfall {
				if m.log.GetLevel() == logging.DebugLevel {
					m.log.Debug("party did not have enough collateral to reach the InitialMargin",
						logging.Order(*order),
						logging.String("market-id", m.GetID()))
				}

				// Rollback transfers in case the order do not
				// trade and do not stay in the book to prevent for margin being
				// locked in the margin account forever
				if nerr := m.collateral.RollbackTransfers(ctx, tr); nerr != nil {
					m.log.Error(
						"Failed to roll back margin transfers for party",
						logging.String("party-id", order.PartyId),
						logging.Error(nerr),
					)
				}
				return ErrMarginCheckInsufficient, nil
			}
			if nerr := m.applyBondPenalty(ctx, order.PartyId, shortfall, asset); nerr != nil {
				m.log.Error("unable to apply bond penalty",
					logging.String("market-id", m.GetID()),
					logging.String("party-id", order.PartyId),
					logging.Error(nerr))
				return nerr, bondPenalty
			}
		}
	}
	if err != nil {
		return err, bondPenalty
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tr}))
	return nil, nil

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
