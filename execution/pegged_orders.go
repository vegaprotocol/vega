package execution

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type PeggedOrders struct {
	currentTime int64
	orders      []*types.Order

	ordersChanged bool
}

func NewPeggedOrders() *PeggedOrders {
	return &PeggedOrders{
		orders: []*types.Order{},
	}
}

func NewPeggedOrdersFromSnapshot(orders []*types.Order) *PeggedOrders {
	return &PeggedOrders{
		orders: orders,
	}
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

func (p *PeggedOrders) OnTimeUpdate(t time.Time) {
	p.currentTime = t.UnixNano()
}

func (p *PeggedOrders) Park(o *types.Order) {
	o.UpdatedAt = p.currentTime
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
	for _, o := range p.orders {
		if o.Party == party /* && o.Status == types.Order_STATUS_PARKED */ {
			o.UpdatedAt = p.currentTime
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

	for _, o := range p.orders {
		if o.Party == party && o.Status == types.OrderStatusParked {
			o.UpdatedAt = p.currentTime
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

	return peggedOrders
}
