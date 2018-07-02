package rest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"vega/api/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"vega/proto"
	"errors"
)

func TestHandlers_Index(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	handlers := Handlers{}
	handlers.Index(context)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "V E G A", w.Body.String())
}

func TestHandlers_CreateOrderWithModelWhenValidReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	order := buildNewOrder()
	orderService := &mocks.OrderService{}
	orderService.On("CreateOrder", context, order).Return(
		true, nil,
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.CreateOrderWithModel(context, order)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "{\"result\":\"success\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

func TestHandlers_CreateOrderWithModelWhenErrorReturnsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	order := buildNewOrder()
	orderService := &mocks.OrderService{}
	orderService.On("CreateOrder", context, order).Return(
		false, errors.New("An expected error"),
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.CreateOrderWithModel(context, order)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t, "{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

func TestHandlers_GetOrdersReturnsSuccessWithModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	context.Request = &http.Request{Method: "GET", URL: &url.URL{Path: "/orders?market=test"}}

	market := "BTC/DEC18"
	limit := uint64(18446744073709551615)
	orderService := &mocks.OrderService{}
	orderService.On("GetOrders", context, market, limit).Return(
		[]msg.Order{
			{Id: "1"},
			{Id: "2"},
		}, nil,
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrders(context)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "{\"orders\":[{\"id\":\"1\"},{\"id\":\"2\"}],\"result\":\"success\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

func TestHandlers_GetOrdersReturnsFailureWhenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)
	context.Request = &http.Request{Method: "GET", URL: &url.URL{Path: "/orders?market=test"}}

	market := "BTC/DEC18"
	limit := uint64(18446744073709551615)

	orderService := &mocks.OrderService{}
	orderService.On("GetOrders", context, market, limit).Return(
		[]msg.Order{}, errors.New("An expected error"),
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrders(context)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t, "{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

func TestHandlers_GetOrdersWithParamsReturnsSuccessWithModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	market := "BTC/TEST"
	limit := uint64(18446744073709551615)

	orderService := &mocks.OrderService{}
	orderService.On("GetOrders", context, market, limit).Return(
		[]msg.Order{
			{Id: "1"},
			{Id: "2"},
		}, nil,
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrdersWithParams(context, market, limit)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "{\"orders\":[{\"id\":\"1\"},{\"id\":\"2\"}],\"result\":\"success\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

func TestHandlers_GetOrdersWithParamsReturnsFailureWhenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	market := "BTC/TEST"
	limit := uint64(18446744073709551615)

	orderService := &mocks.OrderService{}
	orderService.On("GetOrders", context, market, limit).Return(
		nil, errors.New("An expected error"),
	).Once()

	handlers := Handlers{
		OrderService: orderService,
	}

	handlers.GetOrdersWithParams(context, market, limit)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t, "{\"error\":\"An expected error\",\"result\":\"failure\"}", w.Body.String())
	orderService.AssertExpectations(t)
}

// Helpers
func buildNewOrder() msg.Order {
	return msg.Order{
		Id:        "0f2fa7d374415c11054fe7d8dcf04412",
		Market:    "market",
		Party:     "party",
		Side:      0,
		Price:     1,
		Size:      1,
		Remaining: 1,
		Timestamp: 1234567890,
		Type: 1,
	}
}
