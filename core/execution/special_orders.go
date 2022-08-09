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
				cancellation, err := m.matching.RemoveOrderWithStatus(order.ID, types.OrderStatusParked)
				if cancellation == nil || err != nil {
					m.log.Panic("Failure after cancel order from matching engine",
						logging.Order(*order),
						logging.Error(err))
				}

				// Remove it from the party position
				// _ = m.position.UnregisterOrder(ctx, cancellation.Order)
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
				order.Status = types.OrderStatusParked
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
) (_ []*types.Order, enteredAuction bool) {
	updatedOrders := []*types.Order{}

	// Reinsert all the orders
	for _, order := range toSubmitOrders {
		conf, updts, err := m.submitValidatedOrder(ctx, order)
		if err != nil {
			m.log.Debug("could not re-submit a pegged order after repricing",
				logging.MarketID(m.GetID()),
				logging.PartyID(order.Party),
				logging.OrderID(order.ID),
				logging.Error(err))
			// order could not be submitted, it's then been rejected
			// we just completely remove it.
			m.removePeggedOrder(order)
		} else if len(conf.Trades) > 0 {
			m.log.Panic("submitting pegged orders after a reprice should never trade",
				logging.Order(*order))
		}

		if m.as.InAuction() {
			enteredAuction = true
			return
		}

		if err == nil {
			updatedOrders = append(updatedOrders, conf.Order)
		}
		updatedOrders = append(updatedOrders, updts...)
	}

	return updatedOrders, false
}

func (m *Market) repriceAllSpecialOrders(
	ctx context.Context,
	changes uint8,
	orderUpdates []*types.Order,
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

	// now we get the list of all LP orders, and get them out of the book
	lpOrders := m.liquidity.GetAllLiquidityOrders()
	// now we remove them all from the book
	for _, order := range lpOrders {
		// Remove order if any volume remains,
		// otherwise it's already been popped by the matching engine.
		cancellation, err := m.cancelOrder(ctx, order.Party, order.ID)
		if cancellation == nil || err != nil {
			m.log.Panic("could not remove liquidity order from the book",
				logging.Order(*order),
				logging.Error(err))
		}
	}

	// now no lp orders are in the book anymore,
	// we can then just re-submit all pegged orders
	// if we needed to re-submit peggted orders,
	// let's do it now
	var updatedPegged []*types.Order
	if needsPeggedUpdates {
		var enteredAuction bool
		updatedPegged, enteredAuction = m.reSubmitPeggedOrders(ctx, toSubmit)
		if enteredAuction {
			// returning nil will stop reference price moves updates
			return nil
		}
	}

	orderUpdates = append(orderUpdates, parked...)
	orderUpdates = append(orderUpdates, updatedPegged...)

	// now we have all the re-submitted pegged orders and the
	// parked pegged orders from before
	// we can call liquidityUpdate, which is going to give us the
	// actual updates to be done on liquidity orders
	bestBidPrice, bestAskPrice, err := m.getBestStaticPricesDecimal()
	if err != nil {
		m.log.Debug("could not get one of the static mid prices",
			logging.Error(err))
		// we do not return here, we could not get one of the prices eventually
	}

	newOrders, cancels, err := m.liquidity.Update(
		ctx, bestBidPrice, bestAskPrice, m.repriceLiquidityOrder, orderUpdates)
	if err != nil {
		// TODO: figure out if error are really possible there,
		// But I'd think not.
		m.log.Error("could not update liquidity", logging.Error(err))
	}

	return m.updateLPOrders(ctx, lpOrders, newOrders, cancels)
}

func (m *Market) enterAuctionSpecialOrders(
	ctx context.Context,
	updatedOrders []*types.Order,
) []*types.Order {
	// Park all pegged orders
	updatedOrders = append(
		updatedOrders,
		m.parkAllPeggedOrders(ctx)...,
	)

	// we know we enter an auction here,
	// so let's just get the list of all orders, and cancel them
	bestBidPrice, bestAskPrice, err := m.getBestStaticPricesDecimal()
	if err != nil {
		m.log.Debug("could not get one of the static mid prices",
			logging.Error(err))
		// we do not return here, we could not get one of the prices eventually
	}
	newOrders, cancels, err := m.liquidity.Update(
		ctx, bestBidPrice, bestAskPrice, m.repriceLiquidityOrder, updatedOrders)
	if err != nil {
		// TODO: figure out if error are really possible there,
		// But I'd think not.
		m.log.Error("could not update liquidity", logging.Error(err))
	}

	// we are entering an auction, the liquidity engine should always instruct
	// to cancel all orders, and recreating none
	if len(newOrders) > 0 {
		m.log.Panic("liquidity engine instructed to create orders when entering auction",
			logging.MarketID(m.GetID()),
			logging.Int("new-order-count", len(newOrders)))
	}

	// method always return nil anyway
	// TODO: API to be changed someday as we don't need to cancel anything
	// now, we assume that all that were required to be cancelled already are.
	orderUpdates, _ := m.updateAndCreateLPOrders(
		ctx, []*types.Order{}, cancels, []*types.Order{})
	return orderUpdates
}

func (m *Market) updateLPOrders(
	ctx context.Context,
	allOrders []*types.Order,
	submits []*types.Order,
	cancels []*liquidity.ToCancel,
) []*types.Order {
	cancelIDs := map[string]struct{}{}

	// now we gonna map all the all order which
	// where to be cancelled, and just do nothing in
	// those case.
	for _, v := range cancels {
		for _, id := range v.OrderIDs {
			cancelIDs[id] = struct{}{}
		}
	}

	// this is a list of order which a LP distressed
	var (
		distressedOrders  []*types.Order
		distressedParties = map[string]struct{}{}
	)

	// now we iterate over all the orders which
	// were initially cancelled, and remove them
	// from the list if the liquidity engine instructed to
	// cancel them
	var cancelEvts []events.Event
	for _, order := range allOrders {
		if _, ok := distressedParties[order.Party]; ok {
			// party is distressed, not processing
			continue
		}

		// these order were actually cancelled, just send the event
		if _, ok := cancelIDs[order.ID]; ok {
			// cancelEvts = append(cancelEvts, events.NewOrderEvent(ctx, order))
			// these orders were submitted exactly the same before,
			// so there's no reason we would not be able to submit
			// let's panic if an issue happen
		} else {
			// set the status to active again
			order.Status = types.OrderStatusActive
			conf, _, err := m.submitValidatedOrder(ctx, order)
			if conf == nil || err != nil {
				distressedOrders = append(distressedOrders, order)
				distressedParties[order.Party] = struct{}{}
			} else if len(conf.Trades) > 0 {
				m.log.Panic("submitting liquidity orders after a reprice should never trade",
					logging.Order(*order))
			}
		}

		// if an auction has been started, we just break now
		if m.as.InAuction() {
			// enteredAuction = true
			// auctionFrom = i
			// break
			return nil
		}
	}

	// send cancel events
	m.broker.SendBatch(cancelEvts)

	// method always return nil anyway
	// TODO: API to be changed someday as we don't need to cancel anything
	// now, we assume that all that were required to be cancelled already are.
	orderUpdates, _ := m.updateAndCreateLPOrders(
		ctx, submits, []*liquidity.ToCancel{}, distressedOrders)
	return orderUpdates
}
