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
	"code.vegaprotocol.io/vega/core/types"

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
