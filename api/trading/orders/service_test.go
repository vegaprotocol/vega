package orders

import (
	"testing"
	"vega/api/trading/orders/mocks"
	"vega/api/trading/orders/models"
	"github.com/stretchr/testify/assert"
	"context"
)

func TestOrderService_CreateOrder(t *testing.T) {

	var ctx = context.Background()
	var orderService mocks.MockOrderService
	orderService = mocks.MockOrderService{
		ResultSuccess: true,
	}
	
	order := models.NewOrder("d41d8cd98f00b204e9800998ecf8427e", "BTC/DEC18", "PARTY", 1, 1000, 25, 0, 1234567890, 1)
	success, err := orderService.CreateOrder(ctx, order)

	assert.True(t, success)
	assert.Nil(t, err)
}

