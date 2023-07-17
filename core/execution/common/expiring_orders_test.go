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
