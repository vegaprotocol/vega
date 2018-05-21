package rest

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
	"vega/api/mocks"
	"vega/api/trading/orders"
)

func TestIndexHandler_ReturnsExpectedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	handlers := Handlers {}
	handlers.Index(context)

	context.Request, _ = http.NewRequest(http.MethodGet, indexRoute, nil)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "V E G A", w.Body.String())
}

func TestCreateOrderHandler_ReturnsExpectedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	orderService := &mocks.MockOrderService{}
	handlers := Handlers {
		OrderService: orderService,
	}

	var o orders.Order
	o = buildNewOrder()
	handlers.CreateOrderWithModel(context, o)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"{\"result\":\"success\"}", w.Body.String())
}

// Helpers
func buildNewOrder() orders.Order  {
	return orders.NewOrder("market", "party", 0, 1,1, 1, 1234567890, 1)
}
