package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
)

type PeggedOrders struct {
	currentTime int64
	orders      []*types.Order
}

func NewPeggedOrders() *PeggedOrders {
	return &PeggedOrders{
		orders: []*types.Order{},
	}
}

func (p *PeggedOrders) OnTimeUpdate(t time.Time) {
	p.currentTime = t.UnixNano()
}

func (p *PeggedOrders) Park(o *types.Order) {
	o.UpdatedAt = p.currentTime
	o.Status = types.Order_STATUS_PARKED
	o.Price = 0
}

func (p *PeggedOrders) GetByID(id string) *types.Order {
	for _, o := range p.orders {
		if o.Id == id {
			return o
		}
	}
	return nil
}

func (p *PeggedOrders) Add(o *types.Order) {
	p.orders = append(p.orders, o)
}

func (p *PeggedOrders) Remove(o *types.Order) {
	for i, po := range p.orders {
		if po.Id == o.Id {
			// Remove item from slice
			copy(p.orders[i:], p.orders[i+1:])
			p.orders[len(p.orders)-1] = nil
			p.orders = p.orders[:len(p.orders)-1]
			return
		}
	}
}

func (p *PeggedOrders) Amend(amended *types.Order) {
	for i, o := range p.orders {
		if o.Id == amended.Id {
			p.orders[i] = amended
			return
		}
	}
}

func (p *PeggedOrders) RemoveAllForParty(
	ctx context.Context, party string, status types.Order_Status,
) (orders []*types.Order, evts []events.Event) {
	n := 0
	for _, o := range p.orders {
		if o.PartyId == party /* && o.Status == types.Order_STATUS_PARKED */ {
			o.UpdatedAt = p.currentTime
			o.Status = status
			orders = append(orders, o)
			evts = append(evts, events.NewOrderEvent(ctx, o))
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
	ctx context.Context, party string, status types.Order_Status,
) (orders []*types.Order, evts []events.Event) {
	n := 0
	for _, o := range p.orders {
		if o.PartyId == party && o.Status == types.Order_STATUS_PARKED {
			o.UpdatedAt = p.currentTime
			o.Status = status
			orders = append(orders, o)
			evts = append(evts, events.NewOrderEvent(ctx, o))
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
		if order.Status != types.Order_STATUS_PARKED {
			orders = append(orders, order)
		}
	}
	return
}

func (p *PeggedOrders) GetAllParkedForParty(party string) (orders []*types.Order) {
	for _, order := range p.orders {
		if order.PartyId == party && order.Status == types.Order_STATUS_PARKED {
			orders = append(orders, order)
		}
	}
	return
}

func (p *PeggedOrders) GetAllForParty(party string) (orders []*types.Order) {
	for _, order := range p.orders {
		if order.PartyId == party {
			orders = append(orders, order)
		}
	}
	return
}
