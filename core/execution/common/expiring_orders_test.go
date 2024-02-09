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

package common_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/execution/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiringOrders(t *testing.T) {
	t.Run("expire orders ", testExpireOrders)
	t.Run("snapshot ", testExpireOrdersSnapshot)
}

func testExpireOrders(t *testing.T) {
	eo := common.NewExpiringOrders()
	eo.Insert("1", 100)
	eo.Insert("2", 110)
	eo.Insert("3", 140)
	eo.Insert("4", 140)
	eo.Insert("5", 160)
	eo.Insert("6", 170)

	// remove them once
	orders := eo.Expire(140)
	assert.Equal(t, 4, len(orders))
	assert.Equal(t, "1", orders[0])
	assert.Equal(t, "2", orders[1])
	assert.Equal(t, "3", orders[2])
	assert.Equal(t, "4", orders[3])

	// try again to remove to check if they are still there.
	orders = eo.Expire(140)
	assert.Equal(t, 0, len(orders))

	// now try to remove one more
	orders = eo.Expire(160)
	assert.Equal(t, 1, len(orders))
	assert.Equal(t, "5", orders[0])
}

func testExpireOrdersSnapshot(t *testing.T) {
	a := assert.New(t)
	eo := common.NewExpiringOrders()
	a.True(eo.Changed())

	testOrders := getTestOrders()[:6]

	// Test empty
	a.Len(eo.GetState(), 0)

	eo.Insert(testOrders[0].ID, 100)
	eo.Insert(testOrders[1].ID, 110)
	eo.Insert(testOrders[2].ID, 140)
	eo.Insert(testOrders[3].ID, 140)
	eo.Insert(testOrders[4].ID, 160)
	eo.Insert(testOrders[5].ID, 170)
	a.True(eo.Changed())

	testIDs := map[string]struct{}{}
	for _, to := range testOrders {
		testIDs[to.ID] = struct{}{}
	}

	s := eo.GetState()

	newEo := common.NewExpiringOrdersFromState(s)
	a.True(newEo.Changed())
	state := newEo.GetState()
	a.Equal(len(testIDs), len(state))
	for _, o := range state {
		require.NotNil(t, testIDs[o.ID])
	}
}
