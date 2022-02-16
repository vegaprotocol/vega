package execution

import (
	"code.vegaprotocol.io/vega/types"

	"github.com/google/btree"
)

func NewExpiringOrdersFromState(orders []*types.Order) *ExpiringOrders {
	eo := &ExpiringOrders{
		orders:        btree.New(2),
		ordersChanged: true,
	}

	for _, o := range orders {
		eo.Insert(o.ID, o.ExpiresAt)
	}

	return eo
}

func (a ExpiringOrders) Changed() bool {
	return a.ordersChanged
}

func (a *ExpiringOrders) GetState() []*types.Order {
	orders := make([]*types.Order, 0, a.orders.Len())
	a.orders.Ascend(func(item btree.Item) bool {
		oo := item.(*ordersAtTS)
		for _, o := range oo.orders {
			// We don't actually need the entire order to save/restore this state, just the ID and expiry
			// we could consider changing the snapshot protos to reflect this.
			orders = append(orders, &types.Order{
				ID:        o,
				ExpiresAt: oo.ts,
			})
		}
		return true
	})

	a.ordersChanged = false
	return orders
}
