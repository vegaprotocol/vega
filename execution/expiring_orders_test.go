package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpiringOrders(t *testing.T) {
	t.Run("expire orders ", testExpireOrders)
	t.Run("snapshot ", testExpireOrdersSnapshot)
}

func testExpireOrders(t *testing.T) {
	eo := NewExpiringOrders()
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
	eo := NewExpiringOrders()
	a.True(eo.changed())

	testOrders := getTestOrders()[:6]

	// Test empty
	a.Equal([]string{}, eo.GetState())
	a.False(eo.changed())

	eo.Insert(testOrders[0].ID, 100)
	eo.Insert(testOrders[1].ID, 110)
	eo.Insert(testOrders[2].ID, 140)
	eo.Insert(testOrders[3].ID, 140)
	eo.Insert(testOrders[4].ID, 160)
	eo.Insert(testOrders[5].ID, 170)
	a.True(eo.changed())

	testIDs := []string{}
	for _, to := range testOrders {
		testIDs = append(testIDs, to.ID)
	}

	s := eo.GetState()
	a.False(eo.changed())
	a.Equal(testIDs, s)

	newEo := NewExpiringOrdersFromState(testOrders)
	a.True(newEo.changed())
	a.Equal(testIDs, newEo.GetState())
	a.False(newEo.changed())
}
