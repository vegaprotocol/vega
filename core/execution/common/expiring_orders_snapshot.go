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
	"code.vegaprotocol.io/vega/core/types"

	"github.com/google/btree"
)

func NewExpiringOrdersFromState(orders []*types.Order) *ExpiringOrders {
	eo := &ExpiringOrders{
		orders: btree.New(2),
	}

	for _, o := range orders {
		eo.Insert(o.ID, o.ExpiresAt)
	}

	return eo
}

func (a ExpiringOrders) Changed() bool {
	return true
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

	return orders
}
