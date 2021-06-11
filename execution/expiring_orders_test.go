package execution_test

import (
	"testing"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

func TestExpiringOrders(t *testing.T) {
	t.Run("expire orders ", testExpireOrders)
}

func testExpireOrders(t *testing.T) {
	eo := execution.NewExpiringOrders()
	eo.Insert(types.Order{ExpiresAt: 100, Id: "1"})
	eo.Insert(types.Order{ExpiresAt: 110, Id: "2"})
	eo.Insert(types.Order{ExpiresAt: 140, Id: "3"})
	eo.Insert(types.Order{ExpiresAt: 140, Id: "4"})
	eo.Insert(types.Order{ExpiresAt: 160, Id: "5"})
	eo.Insert(types.Order{ExpiresAt: 170, Id: "6"})

	// remove them once
	orders := eo.Expire(140)
	assert.Equal(t, 4, len(orders))
	assert.Equal(t, int64(100), orders[0].ExpiresAt)
	assert.Equal(t, "1", orders[0].Id)
	assert.Equal(t, int64(110), orders[1].ExpiresAt)
	assert.Equal(t, "2", orders[1].Id)
	assert.Equal(t, int64(140), orders[2].ExpiresAt)
	assert.Equal(t, "3", orders[2].Id)
	assert.Equal(t, int64(140), orders[3].ExpiresAt)
	assert.Equal(t, "4", orders[3].Id)

	// try again to remove to check if they are still there.
	orders = eo.Expire(140)
	assert.Equal(t, 0, len(orders))

	// now try to remove one more
	orders = eo.Expire(160)
	assert.Equal(t, 1, len(orders))
	assert.Equal(t, int64(160), orders[0].ExpiresAt)
	assert.Equal(t, "5", orders[0].Id)
}
