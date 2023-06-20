package stoporders

import (
	"log"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/maps"
)

type Pool struct {
	log *logging.Logger
	// map partyId * map orderId * StopOrder
	orders map[string]map[string]*types.StopOrder
	// useful to find back a party from an order
	orderToParty map[string]string
	priced       *PricedStopOrders
	trailing     *TrailingStopOrders
}

func New(log *logging.Logger) *Pool {
	return &Pool{
		log:          log,
		orders:       map[string]map[string]*types.StopOrder{},
		orderToParty: map[string]string{},
		priced:       NewPricedStopOrders(),
		trailing:     NewTrailingStopOrders(),
	}
}

func NewFromProto(log *logging.Logger, p *v1.StopOrders) *Pool {
	pool := New(log)

	for _, porder := range p.StopOrders {
		order := types.NewStopOrderFromProto(porder)

		if party, ok := pool.orders[order.Party]; ok {
			if _, ok := party[order.ID]; ok {
				pool.log.Panic("stop order already exists", logging.String("id", order.ID))
			}
		} else {
			pool.orders[order.Party] = map[string]*types.StopOrder{}
		}

		pool.orders[order.Party][order.ID] = order
		pool.orderToParty[order.ID] = order.Party
	}

	pool.priced = NewPricedStopOrdersFromProto(p.PricedStopOrders)
	pool.trailing = NewTrailingStopOrdersFromProto(p.TrailingStopOrders)

	return pool
}

func (p *Pool) ToProto() *v1.StopOrders {
	out := &v1.StopOrders{}

	for _, v := range p.orders {
		for _, order := range v {
			out.StopOrders = append(out.StopOrders, order.ToProtoEvent())
		}
	}

	sort.Slice(out.StopOrders, func(i, j int) bool {
		return out.StopOrders[i].StopOrder.Id < out.StopOrders[j].StopOrder.Id
	})

	out.PricedStopOrders = p.priced.ToProto()
	out.TrailingStopOrders = p.trailing.ToProto()

	return out
}

func (p *Pool) PriceUpdated(newPrice *num.Uint) (triggered, cancelled []*types.StopOrder) {
	// first update prices and get triggered orders
	ids := append(
		p.priced.PriceUpdated(newPrice),
		p.trailing.PriceUpdated(newPrice)...,
	)

	// first get all the orders which got triggered
	for _, v := range ids {
		pid, ok := p.orderToParty[v]
		if !ok {
			log.Panic("order in tree but not in pool", logging.String("order-id", v))
		}

		// not needed anymore
		delete(p.orderToParty, v)

		orders, ok := p.orders[pid]
		if !ok {
			p.log.Panic("party was expected to have orders but have none",
				logging.String("party-id", pid), logging.String("order-id", v))
		}

		// now we are down to the actual order
		sorder, ok := orders[v]
		if !ok {
			p.log.Panic("party was expected to have an order",
				logging.String("party-id", pid), logging.String("order-id", v))
		}

		sorder.Status = types.StopOrderStatusTriggered
		triggered = append(triggered, sorder)

		// now we can cleanup
		delete(orders, v)
		if len(orders) <= 0 {
			// we can remove the trader altogether
			delete(p.orders, pid)
		}
	}

	// now we get all the OCO oposit to them as they shall
	// be cancelled as well
	for _, v := range triggered[:] {
		if len(v.OCOLinkID) <= 0 {
			continue
		}

		res, err := p.removeWithOCO(v.Party, v.OCOLinkID, false)
		if err != nil || len(res) <= 0 {
			// that should never happen, this mean for some
			// reason that the other side of the OCO has been
			// remove and left the pool in a bad state
			p.log.Panic("other side of the oco missing from the pool",
				logging.Error(err),
				logging.PartyID(v.Party),
				logging.OrderID(v.OCOLinkID))
		}

		// only one order returned here
		res[0].Status = types.StopOrderStatusStopped
		cancelled = append(cancelled, res[0])
	}

	return triggered, cancelled
}

func (p *Pool) Insert(order *types.StopOrder) {
	if party, ok := p.orders[order.Party]; ok {
		if _, ok := party[order.ID]; ok {
			p.log.Panic("stop order already exists", logging.String("id", order.ID))
		}
	} else {
		p.orders[order.Party] = map[string]*types.StopOrder{}
	}

	p.orders[order.Party][order.ID] = order
	p.orderToParty[order.ID] = order.Party
	switch {
	case order.Trigger.IsPrice():
		p.priced.Insert(order.ID, order.Trigger.Price().Clone(), order.Trigger.Direction)
	case order.Trigger.IsTrailingPercenOffset():
		p.trailing.Insert(order.ID, order.Trigger.TrailingPercentOffset(), order.Trigger.Direction)
	}
}

func (p *Pool) Cancel(
	partyID string,
	orderID string, // if empty remove all
) ([]*types.StopOrder, error) {
	orders, err := p.removeWithOCO(partyID, orderID, true)
	if err == nil {
		for _, v := range orders {
			v.Status = types.StopOrderStatusCancelled
		}
	}

	return orders, err
}

func (p *Pool) removeWithOCO(
	partyID string,
	orderID string,
	withOCO bool, // not always necessary in case we are
) ([]*types.StopOrder, error) {
	partyOrders, ok := p.orders[partyID]
	if !ok {
		// this party have no stop orders, move on
		return nil, nil
	}

	// remove a single one and maybe OCO
	if len(orderID) > 0 {
		order, ok := partyOrders[orderID]
		if !ok {
			return nil, ErrStopOrderNotFound
		}

		orders := []*types.StopOrder{order}
		if withOCO && len(order.OCOLinkID) > 0 {
			orders = append(orders, partyOrders[order.OCOLinkID])
		}

		p.remove(orders)

		return orders, nil
	}

	orders := maps.Values(partyOrders)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	p.remove(orders)

	return orders, nil
}

func (p *Pool) remove(orders []*types.StopOrder) {
	for _, order := range orders {
		delete(p.orderToParty, order.ID)
		delete(p.orders[order.Party], order.ID)

		if len(p.orders[order.Party]) <= 0 {
			// no need of this entry anymore
			delete(p.orders, order.Party)
		}

		switch {
		case order.Trigger.IsPrice():
			p.priced.Remove(order.ID)
		case order.Trigger.IsTrailingPercenOffset():
			p.trailing.Remove(order.ID)
		}
	}
}

func (p *Pool) RemoveExpired(orderIDs []string) []*types.StopOrder {
	ordersM := map[string]*types.StopOrder{}

	// first find all orders and add them to the map
	for _, id := range orderIDs {
		order := p.orders[p.orderToParty[id]][id]
		order.Status = types.StopOrderStatusExpired
		ordersM[id] = order

		// once an order is removed, we also remove it's OCO link
		if len(order.OCOLinkID) > 0 {
			// first check if it's not been removed already
			if _, ok := p.orderToParty[order.OCOLinkID]; ok {
				// is the OCO link already mapped
				if _, ok := ordersM[order.OCOLinkID]; !ok {
					ordersM[order.OCOLinkID] = p.orders[p.orderToParty[id]][order.OCOLinkID]
					ordersM[order.OCOLinkID].Status = types.StopOrderStatusExpired
				}
			}
		}
	}

	orders := maps.Values(ordersM)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	p.remove(orders)

	return orders
}
