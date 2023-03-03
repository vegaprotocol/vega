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

//lint:file-ignore U1000 Ignore unused functions

package execution

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) repricePeggedOrders(
	ctx context.Context,
	changes uint8,
) (parked []*types.Order, toSubmit []*types.Order) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "repricePeggedOrders")

	// Go through *all* of the pegged orders and remove from the order book
	// NB: this is getting all of the pegged orders that are unparked in the order book AND all
	// the parked pegged orders.
	allPeggedIDs := m.matching.GetActivePeggedOrderIDs()
	allPeggedIDs = append(allPeggedIDs, m.peggedOrders.GetParkedIDs()...)
	for _, oid := range allPeggedIDs {
		var (
			order *types.Order
			err   error
		)
		if m.peggedOrders.IsParked(oid) {
			order = m.peggedOrders.GetParkedByID(oid)
		} else {
			order, err = m.matching.GetOrderByID(oid)
			if err != nil {
				m.log.Panic("if order is not parked, it should be on the book", logging.OrderID(oid))
			}
		}
		if OrderReferenceCheck(*order).HasMoved(changes) {
			// First if the order isn't parked, then
			// we will just remove if from the orderbook
			if order.Status != types.OrderStatusParked {
				// Remove order if any volume remains,
				// otherwise it's already been popped by the matching engine.
				cancellation, err := m.matching.CancelOrder(order)
				if cancellation == nil || err != nil {
					m.log.Panic("Failure after cancel order from matching engine",
						logging.Order(*order),
						logging.Error(err))
				}

				// Remove it from the party position
				_ = m.position.UnregisterOrder(ctx, order)
			} else {
				// unpark before it's reparked next eventually
				m.peggedOrders.Unpark(order.ID)
			}

			if price, err := m.getNewPeggedPrice(order); err != nil {
				// Failed to reprice, we need to park again
				order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
				order.Status = types.OrderStatusParked
				order.Price = num.UintZero()
				order.OriginalPrice = nil
				m.broker.Send(events.NewOrderEvent(ctx, order))
				parked = append(parked, order)
			} else {
				// Repriced so all good make sure status is correct
				order.Price = price.Clone()
				order.OriginalPrice = price.Clone()
				order.OriginalPrice.Div(order.OriginalPrice, m.priceFactor)
				order.Status = types.OrderStatusActive
				order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
				toSubmit = append(toSubmit, order)
			}
		}
	}

	timer.EngineTimeCounterAdd()

	return parked, toSubmit
}

func (m *Market) reSubmitPeggedOrders(
	ctx context.Context,
	toSubmitOrders []*types.Order,
) ([]*types.Order, map[string]events.MarketPosition) {
	var (
		partiesPos    = map[string]events.MarketPosition{}
		updatedOrders = []*types.Order{}
		evts          = []events.Event{}
	)

	// Reinsert all the orders
	for _, order := range toSubmitOrders {
		m.matching.ReSubmitSpecialOrders(order)
		partiesPos[order.Party] = m.position.RegisterOrder(ctx, order)
		updatedOrders = append(updatedOrders, order)
		evts = append(evts, events.NewOrderEvent(ctx, order))
	}

	// send new order events
	m.broker.SendBatch(evts)

	return updatedOrders, partiesPos
}

func (m *Market) repriceAllSpecialOrders(
	ctx context.Context,
	changes uint8,
	orderUpdates []*types.Order,
	minLpPrice, maxLpPrice *num.Uint,
) []*types.Order {
	if changes == 0 && len(orderUpdates) <= 0 {
		// nothing to do, prices didn't move,
		// no orders have been updated, there's no
		// reason pegged order should get repriced or
		// lp to be differnet than before
		return nil
	}

	// first we get all the pegged orders to be resubmitted with a new price
	var parked, toSubmit []*types.Order
	if changes != 0 {
		parked, toSubmit = m.repricePeggedOrders(ctx, changes)
		for _, topark := range parked {
			m.peggedOrders.Park(topark)
		}
	}

	// just checking if we need to take all lp of the book too
	// normal lp updates would be fine without taking order from the
	// book as no prices would be conlficting
	needsPeggedUpdates := len(parked) > 0 || len(toSubmit) > 0

	// first we save all the LP orders into the liquidity engine so that it can
	// know the history during Update
	m.liquidity.SaveLPOrders()
	defer m.liquidity.ClearLPOrders()

	// now we get the list of all LP orders, and get them out of the book
	lpOrders := m.matching.GetAllLiquidityOrders()
	m.removeLPOrdersFromBook(ctx, lpOrders)

	// now no lp orders are in the book anymore,
	// we can then just re-submit all pegged orders
	// if we needed to re-submit peggted orders,
	// let's do it now
	partiesPos := map[string]events.MarketPosition{}
	if needsPeggedUpdates {
		_, partiesPos = m.reSubmitPeggedOrders(ctx, toSubmit)
	}

	// now we have all the re-submitted pegged orders and the
	// parked pegged orders from before
	// we can call Update, which is going to give us the
	// actual updates to be done on liquidity orders
	newOrders, cancels := m.liquidity.Update(
		ctx, minLpPrice, maxLpPrice, m.repriceLiquidityOrder)

	m.liquidity.ClearLPOrders()
	return m.updateLPOrders(ctx, lpOrders, newOrders, cancels, partiesPos)
}

func (m *Market) enterAuctionSpecialOrders(
	ctx context.Context,
	updatedOrders []*types.Order,
) {
	// first remove all GFN orders from the peg list
	ordersEvts := m.peggedOrders.EnterAuction(ctx)
	m.broker.SendBatch(ordersEvts)

	m.stopAllSpecialOrders(ctx, updatedOrders)
}

func (m *Market) stopAllSpecialOrders(
	ctx context.Context,
	updatedOrders []*types.Order,
) {
	// Park all pegged orders
	updatedOrders = append(
		updatedOrders,
		m.parkAllPeggedOrders(ctx)...,
	)

	// now we just get the list of all LPs to be cancelled
	cancels := m.liquidity.UndeployLPs(ctx, updatedOrders)
	market := m.GetID()

	for _, cancel := range cancels {
		for _, orderID := range cancel.OrderIDs {
			if _, err := m.cancelOrder(ctx, cancel.Party, orderID); err != nil {
				// here we panic, an order which should be in a the market
				// appears not to be. there's either an issue in the liquidity
				// engine and we are trying to remove a non-existing order
				// or the market lost track of the order
				m.log.Panic("unable to amend a liquidity order",
					logging.OrderID(orderID),
					logging.PartyID(cancel.Party),
					logging.MarketID(market),
					logging.Error(err))
			}
		}
	}
}

func (m *Market) updateLPOrders(
	ctx context.Context,
	allOrders []*types.Order,
	submits []*types.Order,
	cancels []*liquidity.ToCancel,
	partiesPos map[string]events.MarketPosition,
) []*types.Order {
	// this is a list of order which a LP distressed
	var (
		orderEvts []events.Event
		cancelIDs = map[string]struct{}{}
		submitIDs = map[string]struct{}{}
		now       = m.timeService.GetTimeNow().UnixNano()
	)

	// now we gonna map all the order which
	// where to be cancelled. Then send events
	// if they are to be cancelled, or do nothing
	// if they are to be submitted again.
	for _, v := range cancels {
		for _, id := range v.OrderIDs {
			cancelIDs[id] = struct{}{}
		}
	}

	// now we gonna map all the all order which
	// where are to be submitted, to avoid cancelling them
	// them submitting them
	for _, v := range submits {
		submitIDs[v.ID] = struct{}{}
	}

	subFn := func(order *types.Order) {
		if order.OriginalPrice == nil {
			order.OriginalPrice = order.Price.Clone()
			order.Price.Mul(order.Price, m.priceFactor)
		}
		// set the status to active again
		order.Status = types.OrderStatusActive
		m.matching.ReSubmitSpecialOrders(order)
		order.Version = 1 // order version never change, just set it explicitly here every time
		partiesPos[order.Party] = m.position.RegisterOrder(ctx, order)
		orderEvts = append(orderEvts, events.NewOrderEvent(ctx, order))
	}

	// now we iterate over all the orders which
	// were initially cancelled, and remove them
	// from the list if the liquidity engine instructed to
	// cancel them, but also the list of all new orders to be created
	for _, order := range allOrders {
		order.UpdatedAt = now

		_, toCancel := cancelIDs[order.ID]
		_, toSubmit := submitIDs[order.ID]
		// these order were actually cancelled, just send the event
		if toCancel {
			if !toSubmit {
				order.Status = types.OrderStatusCancelled
				orderEvts = append(orderEvts, events.NewOrderEvent(ctx, order))
			}
			continue
		}

		subFn(order)
	}

	for _, order := range submits {
		order.UpdatedAt = now
		subFn(order)
	}

	// send cancel events
	m.broker.SendBatch(orderEvts)

	// an ordered list of positions
	var (
		positions     = make([]events.MarketPosition, 0, len(partiesPos))
		marginsBefore = map[string]*num.Uint{}
		id            = m.GetID()
		assetID, _    = m.mkt.GetAsset()
	)
	// now we can check parties positions
	for party, pos := range partiesPos {
		positions = append(positions, pos)
		mar, err := m.collateral.GetPartyMarginAccount(id, party, assetID)
		if err != nil {
			m.log.Panic("party have position without a margin",
				logging.MarketID(id),
				logging.PartyID(party),
			)
		}
		marginsBefore[party] = mar.Balance
	}

	sort.Slice(positions, func(i, j int) bool {
		return positions[i].Party() < positions[j].Party()
	})

	// now we calculate all the new margins
	risks := m.updateMargin(ctx, positions)
	if len(risks) > 0 {
		transfers, closed, bondPenalties, err := m.collateral.MarginUpdate(
			ctx, m.GetID(), risks)
		if err == nil && len(transfers) > 0 {
			evt := events.NewLedgerMovements(ctx, transfers)
			m.broker.Send(evt)
		}

		cancelled := m.applyBondPenaltiesAndLiquidationExcludingPending(
			ctx, bondPenalties, closed, marginsBefore,
		)

		// now ensure we have all parties pending status updated
		for _, v := range positions {
			if m.liquidity.IsLiquidityProvider(v.Party()) {
				if _, ok := cancelled[v.Party()]; !ok {
					// this party LP wasn't cancelled, so it should be now
					// not pending anymore,
					m.liquidity.RemovePending(v.Party())
				}
			}
		}

		_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())

		m.updateLiquidityFee(ctx)
	}

	return []*types.Order{}
}

func (m *Market) applyBondPenaltiesAndLiquidationExcludingPending(
	ctx context.Context,
	bondPenalties []events.Margin,
	closed []events.Margin,
	initialMargins map[string]*num.Uint,
) map[string]struct{} {
	var (
		cancelled    = map[string]struct{}{}
		reallyClosed = []events.Margin{}
	)

	// alright, here we need to go weird over things because we want to find what
	// parties have been considered distressed by the risk / collateral engine BUT
	// for which the LP submission where still pending a first deployment.
	// In which case no bond slashing is being taken, and they shall not be
	// closed as well but the lp submission should only be cancelled.

	// so first we will find all pending which would be closed
	for _, v := range closed {
		if m.liquidity.IsPending(v.Party()) {
			_ = m.cancelPendingLiquidityProvision(
				ctx, v.Party(), initialMargins[v.Party()])
			// adding to the cancelled map to be returned later
			cancelled[v.Party()] = struct{}{}
			continue
		}

		reallyClosed = append(reallyClosed, v)
	}

	// now we can apply the bond slashing, avoiding parties which were
	// pending previously
	for _, bp := range bondPenalties {
		// first short circuit if the node got cancelled
		// party was already cancelled as pending, no penalty for this bois
		if _, ok := cancelled[bp.Party()]; ok {
			continue
		}

		// now we also short circuit if the party wasn't closed but still
		// add bon penalty on first submission
		if m.liquidity.IsPending(bp.Party()) {
			_ = m.cancelPendingLiquidityProvision(
				ctx, bp.Party(), initialMargins[bp.Party()])
			// adding to the cancelled map to be returned later
			cancelled[bp.Party()] = struct{}{}
			continue
		}

		transfers, err := m.bondSlashing(ctx, bp)
		if err != nil {
			m.log.Error("Failed to perform bond slashing", logging.Error(err))
		}
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	// now we can handle the liquidated parties
	if len(reallyClosed) > 0 {
		// now we can had them to the cancelled map
		// as we don't need to use it anymore apart for returning
		for _, v := range reallyClosed {
			cancelled[v.Party()] = struct{}{}
		}

		// _, err := m.resolveClosedOutParties(ctx, reallyClosed)
		// if err != nil {
		// 	m.log.Error("unable to closed out parties",
		// 		logging.String("market-id", m.GetID()),
		// 		logging.Error(err))
		// }
	}

	return cancelled
}

func (m *Market) removeLPOrdersFromBook(ctx context.Context, lpOrders []*types.Order) {
	// now we remove them all from the book
	for _, order := range lpOrders {
		// Just call delete, not status will be set for now.
		cancellation, err := m.matching.DeleteOrder(order)
		if cancellation == nil || err != nil {
			m.log.Panic("could not remove liquidity order from the book",
				logging.Order(*order),
				logging.Error(err))
		}

		order.Status = types.OrderStatusCancelled

		// remove order from the position
		_ = m.position.UnregisterOrder(ctx, order)
	}
}
