package execution_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/execution"

	"github.com/stretchr/testify/assert"
)

func TestExpiringOrders(t *testing.T) {
	t.Run("expire orders ", testExpireOrders)
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
