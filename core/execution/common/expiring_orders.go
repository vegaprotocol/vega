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

package common

import (
	"github.com/google/btree"
)

type ExpiringOrders struct {
	orders *btree.BTree
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
		orders: btree.New(2),
	}
}

func (a *ExpiringOrders) GetExpiryingOrderCount() int {
	result := a.orders.Len()
	return result
}

func (a *ExpiringOrders) Insert(
	orderID string, ts int64,
) {
	item := &ordersAtTS{ts: ts}
	if item := a.orders.Get(item); item != nil {
		item.(*ordersAtTS).orders = append(item.(*ordersAtTS).orders, orderID)
		return
	}
	item.orders = []string{orderID}
	a.orders.ReplaceOrInsert(item)
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

	return orders
}
