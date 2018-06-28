package rest

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"vega/api/trading/orders/mocks"
	"vega/api/trading/orders/models"
	"net/url"
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

func TestHandlers_CreateOrderWithModelWhenValidReturnsSuccess(t *testing.T) {
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

	var o models.Order
	o = buildNewOrder()
	handlers.CreateOrderWithModel(context, o)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"{\"result\":\"success\"}", w.Body.String())
}

func TestHandlers_CreateOrderWithModelWhenErrorReturnsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	orderService := &mocks.MockOrderService{
		ResultSuccess: false,
		ResultError: errors.New("An expected error"),
	}
	order := buildNewOrder()
	handlers := Handlers {
		OrderService: orderService,
	}

	handlers.CreateOrderWithModel(context, order)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
}

func TestHandlers_GetOrdersReturnsSuccessWithModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	context.Request = &http.Request{ Method: "GET", URL: &url.URL{ Path: "/orders?market=test"}}
	orderService := &mocks.MockOrderService{
		ResultSuccess: true,
		ResultOrders: []models.Order{
			{ID: "1"},
			{ID: "2"},
		},
	}
	handlers := Handlers{
		OrderService: orderService,
	}
	
	handlers.GetOrders(context)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "{\"orders\":[{\"id\":\"1\",\"market\":\"\",\"party\":\"\",\"side\":0,\"price\":0,\"size\":0,\"remaining\":0,\"timestamp\":0,\"type\":0},{\"id\":\"2\",\"market\":\"\",\"party\":\"\",\"side\":0,\"price\":0,\"size\":0,\"remaining\":0,\"timestamp\":0,\"type\":0}],\"result\":\"success\"}", w.Body.String())
}


func TestHandlers_GetOrdersReturnsFailureWhenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	context.Request = &http.Request{ Method: "GET", URL: &url.URL{ Path: "/orders?market=test"}}
	orderService := &mocks.MockOrderService{
		ResultSuccess: false,
		ResultError: errors.New("An expected error"),
		ResultOrders: []models.Order{ },
	}
	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrders(context)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
}


func TestHandlers_GetOrdersWithParamsReturnsSuccessWithModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	orderService := &mocks.MockOrderService{
		ResultSuccess: true,
		ResultOrders: []models.Order{
			{ID: "1"},
			{ID: "2"},
		},
	}
	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrdersWithParams(context, "BTC/TEST")

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "{\"orders\":[{\"id\":\"1\",\"market\":\"\",\"party\":\"\",\"side\":0,\"price\":0,\"size\":0,\"remaining\":0,\"timestamp\":0,\"type\":0},{\"id\":\"2\",\"market\":\"\",\"party\":\"\",\"side\":0,\"price\":0,\"size\":0,\"remaining\":0,\"timestamp\":0,\"type\":0}],\"result\":\"success\"}", w.Body.String())
}

func TestHandlers_GetOrdersWithParamsReturnsFailureWhenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	orderService := &mocks.MockOrderService{
		ResultSuccess: false,
		ResultError: errors.New("An expected error"),
		ResultOrders: []models.Order{ },
	}
	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrdersWithParams(context, "BTC/TEST")

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
}


// Helpers
func buildNewOrder() models.Order  {
	return models.NewOrder("0f2fa7d374415c11054fe7d8dcf04412", "market", "party", 0, 1,1, 1, 1234567890, 1)
}