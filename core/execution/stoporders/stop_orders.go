package stoporders

import (
	"log"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
		priced:       &PricedStopOrders{},
		trailing:     &TrailingStopOrders{},
	}
}

func (p *Pool) PriceUpdated(newPrice *num.Uint) []*types.StopOrder {
	// first update prices and get triggered orders
	ids := append(
		p.priced.PriceUpdated(newPrice),
		p.trailing.PriceUpdated(newPrice)...,
	)

	stopOrders := []*types.StopOrder{}

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

		stopOrders = append(stopOrders, sorder)

		// now we can cleanup
		delete(orders, v)
		if len(orders) <= 0 {
			// we can remove the trader altogether
			delete(p.orders, pid)
		}
	}

	// now we get all the OCO oposited to them as they shall
	// be cancelled now
	for _, v := range stopOrders[:] {
		res, err := p.Remove(v.Party, v.OCOLinkID)
		if err != nil {
			// that should never happen, this mean for some
			// reason that the other side of the OCO has been
			// remove and left the pool in a bad state
			p.log.Panic("other side of the oco missing from the pool",
				logging.Error(err),
				logging.PartyID(v.Party),
				logging.OrderID(v.OCOLinkID))
		}

		// only one order returned here
		stopOrders = append(stopOrders, res[0])

	}

	return stopOrders
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

func (p *Pool) Remove(
	partyID string,
	orderID string, // if empty remove all
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
		if len(order.OCOLinkID) > 0 {
			orders = append(orders, partyOrders[order.OCOLinkID])
		}

		p.removeInner(orders)

		return orders, nil
	}

	orders := maps.Values(partyOrders)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	p.removeInner(orders)

	return orders, nil
}

func (p *Pool) removeInner(orders []*types.StopOrder) {
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
		ordersM[id] = order

		// once an order is removed, we also remove it's OCO link
		if len(order.OCOLinkID) > 0 {
			// is the OCO link already mapped
			if _, ok := ordersM[order.OCOLinkID]; !ok {
				ordersM[order.OCOLinkID] = p.orders[p.orderToParty[id]][order.OCOLinkID]
			}
		}
	}

	orders := maps.Values(ordersM)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	p.removeInner(orders)

	return orders
}
