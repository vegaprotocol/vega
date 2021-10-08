package execution

import (
	"code.vegaprotocol.io/vega/types"
	"github.com/google/btree"
)

type ExpiringOrders struct {
	orders        *btree.BTree
	ordersChanged bool
}

type ordersAtTS struct {
	ts int64
	// order IDs
	orders []string
}

func (a *ordersAtTS) Less(b btree.Item) bool {
	return a.ts < b.(*ordersAtTS).ts
}

func NewExpiringOrders() *ExpiringOrders {
	return &ExpiringOrders{
		orders:        btree.New(2),
		ordersChanged: true,
	}
}

func (a ExpiringOrders) changed() bool {
	return a.ordersChanged
}

func (a ExpiringOrders) GetState() []string {
	orders := make([]string, 0, a.orders.Len())
	a.orders.Ascend(func(item btree.Item) bool {
		orders = append(orders, item.(*ordersAtTS).orders...)
		return true
	})

	a.ordersChanged = false

	return orders
}

func (a *ExpiringOrders) RestoreState(orders []*types.Order) {
	for _, o := range orders {
		a.Insert(o.ID, o.ExpiresAt)
	}
}

func (a *ExpiringOrders) GetExpiryingOrderCount() int {
	result := a.orders.Len()
	return result
}

func (a *ExpiringOrders) Insert(
	orderID string, ts int64) {
	item := &ordersAtTS{ts: ts}
	if item := a.orders.Get(item); item != nil {
		item.(*ordersAtTS).orders = append(item.(*ordersAtTS).orders, orderID)
		a.ordersChanged = true
		return
	}
	item.orders = []string{orderID}
	a.orders.ReplaceOrInsert(item)
	a.ordersChanged = true
}

func (a *ExpiringOrders) RemoveOrder(expiresAt int64, orderID string) bool {
	item := &ordersAtTS{ts: expiresAt}
	if item := a.orders.Get(item); item != nil {
		oat := item.(*ordersAtTS)
		for i := 0; i < len(oat.orders); i++ {
			if oat.orders[i] == orderID {
				oat.orders = oat.orders[:i+copy(oat.orders[i:], oat.orders[i+1:])]

				// if the slice is empty, remove the parent container
				if len(oat.orders) == 0 {
					a.orders.Delete(item)
					a.ordersChanged = true
				}
				return true
			}
		}
	}
	return false
}

func (a *ExpiringOrders) Expire(ts int64) []string {
	if a.orders.Len() == 0 {
		return nil
	}
	orders := []string{}
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

	if len(toDelete) > 0 {
		a.ordersChanged = true
	}

	return orders
}
