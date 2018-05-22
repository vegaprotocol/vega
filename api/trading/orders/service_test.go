package orders

import (
	"testing"
	"vega/api/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRpcOrderService_CreateOrder(t *testing.T) {

	var orderService mocks.MockOrderService
	order := NewOrder("BTC/DEC18", "PARTY", 1, 1000, 25, 0, 1234567890, 1)
	success, err := orderService.CreateOrder(order)

	assert.True(t, success)
	assert.Nil(t, err)
}

