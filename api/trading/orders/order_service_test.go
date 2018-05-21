package orders

import (
	"testing"
	"vega/api/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRpcOrderService_CreateOrder(t *testing.T) {

	var orderService mocks.MockOrderService

	success, err := orderService.CreateOrder("BTC/DEC18", "US", 0, 1000, 10)

	assert.True(t, success)
	assert.Nil(t, err)
}

