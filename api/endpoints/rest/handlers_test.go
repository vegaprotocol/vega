package rest

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
	"vega/api/mocks"
	"vega/api/trading/orders"
	"github.com/pkg/errors"
)

func TestHandlers_Index(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	handlers := Handlers {}
	handlers.Index(context)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "V E G A", w.Body.String())
}

func TestHandlers_CreateOrderWithModel_ValidReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	orderService := &mocks.MockOrderService{
		ResultSuccess: true,
		ResultError: nil,
	}
	
	handlers := Handlers {
		OrderService: orderService,
	}

	var o orders.Order
	o = buildNewOrder()
	handlers.CreateOrderWithModel(context, o)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"{\"result\":\"success\"}", w.Body.String())
}

func TestHandlers_CreateOrderWithModel_ErrorReturnsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	orderService := &mocks.MockOrderService{
		ResultSuccess: false,
		ResultError: errors.New("An expected error"),
	}
	
	handlers := Handlers {
		OrderService: orderService,
	}

	var o orders.Order
	o = buildNewOrder()
	handlers.CreateOrderWithModel(context, o)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
}

// Helpers
func buildNewOrder() orders.Order  {
	return orders.NewOrder("market", "party", 0, 1,1, 1, 1234567890, 1)
}