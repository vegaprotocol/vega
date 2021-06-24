//lint:file-ignore U1000 Ignore unused functions

package execution

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/types"
)

func (m *Market) repricePeggedOrders(
	ctx context.Context,
	changes uint8,
) (parked []*types.Order, toSubmit []*types.Order) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "repricePeggedOrders")

	// Go through all the pegged orders and remove from the order book
	for _, order := range m.peggedOrders.orders {
		if OrderReferenceCheck(*order).HasMoved(changes) {
			// First if the order isn't parked, then
			// we will just remove if from the orderbook
			if order.Status != types.Order_STATUS_PARKED {
				// Remove order if any volume remains,
				// otherwise it's already been popped by the matching engine.
				cancellation, err := m.matching.CancelOrder(order)
				if cancellation == nil || err != nil {
					m.log.Panic("Failure after cancel order from matching engine",
						logging.Order(*order),
						logging.Error(err))
				}

				// Remove it from the trader position
				_ = m.position.UnregisterOrder(order)
			}

			if price, err := m.getNewPeggedPrice(order); err != nil {
				// Failed to reprice, if we are parked we do nothing,
				// if not parked we need to park
				if order.Status != types.Order_STATUS_PARKED {
					order.UpdatedAt = m.currentTime.UnixNano()
					order.Status = types.Order_STATUS_PARKED
					order.Price = 0
					m.broker.Send(events.NewOrderEvent(ctx, order))
					parked = append(parked, order)
				}
			} else {
				// Repriced so all good make sure status is correct
				order.Price = price
				order.Status = types.Order_STATUS_PARKED
				toSubmit = append(toSubmit, order)
			}

		}
	}

	timer.EngineTimeCounterAdd()
	return
}

func (m *Market) reSubmitPeggedOrders(
	ctx context.Context,
	toSubmitOrders []*types.Order,
) []*types.Order {
	updatedOrders := []*types.Order{}

	// Reinsert all the orders
	for _, order := range toSubmitOrders {
		conf, updts, err := m.submitValidatedOrder(ctx, order)
		if err != nil {
			m.log.Debug("could not re-submit a pegged order after repricing",
				logging.MarketID(m.GetID()),
				logging.PartyID(order.PartyId),
				logging.OrderID(order.Id),
				logging.Error(err))
			// order could not be submitted, it's then been rejected
			// we just completely remove it.
			m.removePeggedOrder(order)
		} else if len(conf.Trades) > 0 {
			m.log.Panic("submitting pegged orders after a reprice should never trade",
				logging.Order(*order))
		}
		if err == nil {
			updatedOrders = append(updatedOrders, conf.Order)
		}
		updatedOrders = append(updatedOrders, updts...)
	}

	return updatedOrders
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
		cancellation, err := m.matching.CancelOrder(order)
		if cancellation == nil || err != nil {
			m.log.Panic("could not remove liquidity order from the book",
				logging.Order(*order),
				logging.Error(err))
		}

		// Remove it from the trader position
		_ = m.position.UnregisterOrder(order)
	}

	// now no lp orders are in the book anymore,
	// we can then just re-submit all pegged orders
	// if we needed to re-submit peggted orders,
	// let's do it now
	var updatedPegged []*types.Order
	if needsPeggedUpdates {
		updatedPegged = m.reSubmitPeggedOrders(ctx, toSubmit)
	}

	orderUpdates = append(orderUpdates, parked...)
	orderUpdates = append(orderUpdates, updatedPegged...)

	// now we have all the re-submitted pegged orders and the
	// parked pegged orders from before
	// we can call liquidityUpdate, which is going to give us the
	// actual updates to be done on liquidity orders
	bestBidPrice, bestAskPrice, err := m.getBestStaticPrices()
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
	bestBidPrice, bestAskPrice, err := m.getBestStaticPrices()
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
		orders  []*types.Order
		parties = map[string]struct{}{}
	)

	// now we iterate over all the orders which
	// were initially cancelled, and remove them
	// from the list if the liquidity engine instructed to
	// cancel them
	var cancelEvts []events.Event
	for _, order := range allOrders {
		if _, ok := parties[order.PartyId]; ok {
			// party is distressed, not processing
			continue
		}

		// these order were actually cancelled, just send the event
		if _, ok := cancelIDs[order.Id]; ok {
			cancelEvts = append(cancelEvts, events.NewOrderEvent(ctx, order))
			// these orders were submitted exactly the same before,
			// so there's no reason we would not be able to submit
			// let's panic if an issue happen
		} else {
			// set the status to active again
			order.Status = types.Order_STATUS_ACTIVE
			conf, _, err := m.submitValidatedOrder(ctx, order)
			if conf == nil || err != nil {
				orders = append(orders, order)
				parties[order.PartyId] = struct{}{}
			} else if len(conf.Trades) > 0 {
				m.log.Panic("submitting liquidity orders after a reprice should never trade",
					logging.Order(*order))
			}
		}
	}

	// send cancel events
	m.broker.SendBatch(cancelEvts)

	// method always return nil anyway
	// TODO: API to be changed someday as we don't need to cancel anything
	// now, we assume that all that were required to be cancelled already are.
	orderUpdates, _ := m.updateAndCreateLPOrders(
		ctx, submits, []*liquidity.ToCancel{}, orders)
	return orderUpdates
}
