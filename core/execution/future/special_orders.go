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

package future

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
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
		if common.OrderReferenceCheck(*order).HasMoved(changes) {
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
) []*types.Order {
	if changes == 0 && len(orderUpdates) <= 0 {
		// nothing to do, prices didn't move,
		// no orders have been updated, there's no
		// reason pegged order should get repriced
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

	needsPeggedUpdates := len(parked) > 0 || len(toSubmit) > 0

	if !needsPeggedUpdates && len(toSubmit) < 1 {
		return nil
	}

	updatedOrders, partiesPos := m.reSubmitPeggedOrders(ctx, toSubmit)
	risks, _, _ := m.updateMargins(ctx, partiesPos)
	if len(risks) > 0 {
		transfers, distressed, _, err := m.collateral.MarginUpdate(
			ctx, m.GetID(), risks)
		if err == nil && len(transfers) > 0 {
			evt := events.NewLedgerMovements(ctx, transfers)
			m.broker.Send(evt)
		}
		for _, p := range distressed {
			distressedParty := p.Party()
			for _, o := range updatedOrders {
				if o.Party == distressedParty && o.Status == types.OrderStatusActive {
					// cancel only the pegged orders, the reset will get picked up during regular closeout flow if need be
					_, err := m.cancelOrder(ctx, distressedParty, o.ID)
					if err != nil {
						m.log.Panic("Failed to cancel order",
							logging.Error(err),
							logging.String("OrderID", o.ID))
					}
				}
			}
		}
	}

	return updatedOrders
}

func (m *Market) updateMargins(ctx context.Context, partiesPos map[string]events.MarketPosition) ([]events.Risk, []events.MarketPosition, map[string]*num.Uint) {
	// an ordered list of positions
	var (
		positions     = make([]events.MarketPosition, 0, len(partiesPos))
		marginsBefore = map[string]*num.Uint{}
		id            = m.GetID()
	)
	// now we can check parties positions
	for party, pos := range partiesPos {
		positions = append(positions, pos)
		mar, err := m.collateral.GetPartyMarginAccount(id, party, m.settlementAsset)
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
	return m.updateMargin(ctx, positions), positions, marginsBefore
}

func (m *Market) enterAuctionSpecialOrders(ctx context.Context) {
	// First remove all GFN orders from the peg list.
	ordersEvts := m.peggedOrders.EnterAuction(ctx)
	m.broker.SendBatch(ordersEvts)

	// Park all pegged orders
	m.parkAllPeggedOrders(ctx)
}
