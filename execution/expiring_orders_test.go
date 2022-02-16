package execution_test

import (
	"testing"

	"code.vegaprotocol.io/vega/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiringOrders(t *testing.T) {
	t.Run("expire orders ", testExpireOrders)
	t.Run("snapshot ", testExpireOrdersSnapshot)
}

func testExpireOrders(t *testing.T) {
	eo := execution.NewExpiringOrders()
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
	eo := execution.NewExpiringOrders()
	a.True(eo.Changed())

	testOrders := getTestOrders()[:6]

	// Test empty
	a.Len(eo.GetState(), 0)
	a.False(eo.Changed())

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
	a.False(eo.Changed())

	newEo := execution.NewExpiringOrdersFromState(s)
	a.True(newEo.Changed())
	state := newEo.GetState()
	a.Equal(len(testIDs), len(state))
	for _, o := range state {
		require.NotNil(t, testIDs[o.ID])
	}
	a.False(newEo.Changed())
}
