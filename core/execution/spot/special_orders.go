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

package spot

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) repricePeggedOrders(ctx context.Context, changes uint8) (parked []*types.Order, toSubmit []*types.Order) {
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
		if common.OrderReferenceCheck(*order).HasMoved(changes) {
			// First if the order isn't parked, then
			// we will just remove if from the orderbook
			if order.Status != types.OrderStatusParked {
				// Remove order if any volume remains,
				// otherwise it's already been popped by the matching engine.
				m.releaseOrderFromHoldingAccount(ctx, order.ID, order.Party, order.Side)
				cancellation, err := m.matching.CancelOrder(order)
				if cancellation == nil || err != nil {
					m.log.Panic("Failure after cancel order from matching engine",
						logging.Order(*order),
						logging.Error(err))
				}
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

func (m *Market) reSubmitPeggedOrders(ctx context.Context, toSubmitOrders []*types.Order) []*types.Order {
	var (
		updatedOrders = []*types.Order{}
		evts          = []events.Event{}
	)

	// Reinsert all the orders
	for _, order := range toSubmitOrders {
		if err := m.checkSufficientFunds(order.Party, order.Side, order.Price, order.TrueRemaining(), order.PeggedOrder != nil); err != nil {
			order.Status = types.OrderStatusStopped
			m.removePeggedOrder(order)
			evts = append(evts, events.NewOrderEvent(ctx, order))
			continue
		}
		m.transferToHoldingAccount(ctx, order)
		m.matching.ReSubmitSpecialOrders(order)
		updatedOrders = append(updatedOrders, order)
		evts = append(evts, events.NewOrderEvent(ctx, order))
	}

	// send new order events
	m.broker.SendBatch(evts)

	return updatedOrders
}

func (m *Market) repriceAllSpecialOrders(
	ctx context.Context,
	changes uint8,
) {
	if changes == 0 {
		// nothing to do, prices didn't move,
		// no orders have been updated, there's no
		// reason pegged order should get repriced or
		// lp to be differnet than before
		return
	}

	// first we get all the pegged orders to be resubmitted with a new price
	var parked, toSubmit []*types.Order
	if changes != 0 {
		parked, toSubmit = m.repricePeggedOrders(ctx, changes)
		for _, topark := range parked {
			m.peggedOrders.Park(topark)
		}
	}

	// if we needed to re-submit pegged orders,
	// let's do it now
	if len(toSubmit) > 0 {
		m.reSubmitPeggedOrders(ctx, toSubmit)
	}
}

func (m *Market) enterAuctionSpecialOrders(ctx context.Context) {
	// first remove all GFN orders from the peg list
	ordersEvts := m.peggedOrders.EnterAuction(ctx)
	m.broker.SendBatch(ordersEvts)
	m.parkAllPeggedOrders(ctx)
}
