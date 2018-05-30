package orders

import (
	"testing"
	"vega/api/trading/orders/mocks"
	"vega/api/trading/orders/models"
	"github.com/stretchr/testify/assert"
)

func TestRpcOrderService_CreateOrder(t *testing.T) {

	var orderService mocks.MockOrderService

	orderService = mocks.MockOrderService{
		ResultSuccess: true,
	}
	
	order := models.NewOrder("BTC/DEC18", "PARTY", 1, 1000, 25, 0, 1234567890, 1)
	success, err := orderService.CreateOrder(order)

	assert.True(t, success)
	assert.Nil(t, err)
}

