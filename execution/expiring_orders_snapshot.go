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

func (a *ExpiringOrders) GetState() []string {
	orders := make([]string, 0, a.orders.Len())
	a.orders.Ascend(func(item btree.Item) bool {
		orders = append(orders, item.(*ordersAtTS).orders...)
		return true
	})

	a.ordersChanged = false

	return orders
}
