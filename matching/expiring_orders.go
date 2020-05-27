package matching

import (
	types "code.vegaprotocol.io/vega/proto"

	"github.com/google/btree"
)

type ExpiringOrders struct {
	orders *btree.BTree
}

type ordersAtTS struct {
	ts     int64
	orders []types.Order
}

func (a *ordersAtTS) Less(b btree.Item) bool {
	return a.ts < b.(*ordersAtTS).ts
}

func NewExpiringOrders() *ExpiringOrders {
	return &ExpiringOrders{
		orders: btree.New(2),
	}
}

func (a *ExpiringOrders) Insert(order types.Order) {
	item := &ordersAtTS{ts: order.ExpiresAt}
	if item := a.orders.Get(item); item != nil {
		item.(*ordersAtTS).orders = append(item.(*ordersAtTS).orders, order)
		return
	}
	item.orders = []types.Order{order}
	a.orders.ReplaceOrInsert(item)
}

func (a *ExpiringOrders) RemoveOrder(order types.Order) bool {
	item := &ordersAtTS{ts: order.ExpiresAt}
	if item := a.orders.Get(item); item != nil {
		oat := item.(*ordersAtTS)
		for i := 0; i < len(oat.orders); i++ {
			if oat.orders[i].Id == order.Id {
				oat.orders = oat.orders[:i+copy(oat.orders[i:], oat.orders[i+1:])]

				// if the slice is empty, remove the parent container
				if len(oat.orders) == 0 {
					a.orders.Delete(item)
				}
				return true
			}
		}
	}
	return false
}

func (a *ExpiringOrders) Expire(ts int64) []types.Order {
	if a.orders.Len() == 0 {
		return nil
	}
	orders := []types.Order{}
	toDelete := []int64{}
	item := &ordersAtTS{ts: ts + 1}
	a.orders.AscendLessThan(item, func(i btree.Item) bool {
		if ts < i.(*ordersAtTS).ts {
			return false
		}
		orders = append(orders, i.(*ordersAtTS).orders...)
		toDelete = append(toDelete, i.(*ordersAtTS).ts)
		return true
	})

	for _, v := range toDelete {
		item.ts = v
		a.orders.Delete(item)
	}

	return orders
}
