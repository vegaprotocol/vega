// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
