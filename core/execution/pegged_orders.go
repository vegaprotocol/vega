// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package execution

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
)

type PeggedOrders struct {
	timeService TimeService
	orders      []*types.Order

	ordersChanged bool
}

func NewPeggedOrders(ts TimeService) *PeggedOrders {
	return &PeggedOrders{
		timeService: ts,
		orders:      []*types.Order{},
	}
}

func NewPeggedOrdersFromSnapshot(orders []*types.Order, tm TimeService) *PeggedOrders {
	return &PeggedOrders{
		timeService: tm,
		orders:      orders,
	}
}

// ReconcileWithOrderBook ensures that any pegged orders that are on the book point to the same
// underlying value.
func (p *PeggedOrders) ReconcileWithOrderBook(orderbook *matching.CachedOrderBook) error {
	newPeggedOrders := make([]*types.Order, 0, len(p.orders))
	for _, o := range p.orders {
		if o.Status == types.OrderStatusParked {
			newPeggedOrders = append(newPeggedOrders, o)
			continue
		}

		order, err := orderbook.GetOrderByID(o.ID)
		if err != nil {
			return err // if its not parked it should be on the book
		}
		newPeggedOrders = append(newPeggedOrders, order)
	}
	p.orders = newPeggedOrders
	return nil
}

func (p *PeggedOrders) Changed() bool {
	return p.ordersChanged
}

func (p *PeggedOrders) GetState() []*types.Order {
	ordersCopy := make([]*types.Order, 0, len(p.orders))
	for _, o := range p.orders {
		ordersCopy = append(ordersCopy, o.Clone())
	}

	p.ordersChanged = false

	return ordersCopy
}

func (p *PeggedOrders) Park(o *types.Order) {
	o.UpdatedAt = p.timeService.GetTimeNow().UnixNano()
	o.Status = types.OrderStatusParked
	o.Price = num.Zero()
	o.OriginalPrice = num.Zero()

	p.ordersChanged = true
}

func (p *PeggedOrders) GetByID(id string) *types.Order {
	for _, o := range p.orders {
		if o.ID == id {
			return o
		}
	}
	return nil
}

func (p *PeggedOrders) Add(o *types.Order) {
	p.orders = append(p.orders, o)
	p.ordersChanged = true
}

func (p *PeggedOrders) Remove(o *types.Order) {
	for i, po := range p.orders {
		if po.ID == o.ID {
			// Remove item from slice
			copy(p.orders[i:], p.orders[i+1:])
			p.orders[len(p.orders)-1] = nil
			p.orders = p.orders[:len(p.orders)-1]
			p.ordersChanged = true
			return
		}
	}
}

func (p *PeggedOrders) Amend(amended *types.Order) {
	for i, o := range p.orders {
		if o.ID == amended.ID {
			p.orders[i] = amended
			p.ordersChanged = true
			return
		}
	}
}

func (p *PeggedOrders) RemoveAllForParty(
	ctx context.Context, party string, status types.OrderStatus,
) (orders []*types.Order, evts []events.Event) {
	n := 0
	now := p.timeService.GetTimeNow().UnixNano()

	for _, o := range p.orders {
		if o.Party == party /* && o.Status == types.Order_STATUS_PARKED */ {
			o.UpdatedAt = now
			o.Status = status
			orders = append(orders, o)
			evts = append(evts, events.NewOrderEvent(ctx, o))
			p.ordersChanged = true
			continue
		}
		// here we insert back in the slice
		p.orders[n] = o
		n++
	}
	p.orders = p.orders[:n]
	return
}

func (p *PeggedOrders) RemoveAllParkedForParty(
	ctx context.Context, party string, status types.OrderStatus,
) (orders []*types.Order, evts []events.Event) {
	n := 0
	now := p.timeService.GetTimeNow().UnixNano()

	for _, o := range p.orders {
		if o.Party == party && o.Status == types.OrderStatusParked {
			o.UpdatedAt = now
			o.Status = status
			orders = append(orders, o)
			evts = append(evts, events.NewOrderEvent(ctx, o))
			p.ordersChanged = true
			continue
		}
		// here we insert back in the slice
		p.orders[n] = o
		n++
	}
	p.orders = p.orders[:n]
	return
}

func (p *PeggedOrders) GetAllActiveOrders() (orders []*types.Order) {
	for _, order := range p.orders {
		if order.Status != types.OrderStatusParked {
			orders = append(orders, order)
		}
	}
	return
}

func (p PeggedOrders) GetAll() []*types.Order {
	return p.orders
}

func (p *PeggedOrders) GetAllParkedForParty(party string) (orders []*types.Order) {
	for _, order := range p.orders {
		if order.Party == party && order.Status == types.OrderStatusParked {
			orders = append(orders, order)
		}
	}
	return
}

func (p *PeggedOrders) GetAllForParty(party string) (orders []*types.Order) {
	for _, order := range p.orders {
		if order.Party == party {
			orders = append(orders, order)
		}
	}
	return
}

func (p *PeggedOrders) Settled() []*types.Order {
	// now we can remove the pegged orders too
	peggedOrders := make([]*types.Order, 0, len(p.orders))
	for _, v := range p.orders {
		if v.Status == types.OrderStatusParked {
			order := v.Clone()
			order.Status = types.OrderStatusStopped
			peggedOrders = append(peggedOrders, order)
		}
	}
	sort.Slice(peggedOrders, func(i, j int) bool {
		return peggedOrders[i].ID < peggedOrders[j].ID
	})

	p.orders = nil

	return peggedOrders
}
