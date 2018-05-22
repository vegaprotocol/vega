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

func TestIndexHandler_ReturnsExpectedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	handlers := Handlers {}
	handlers.Index(context)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "V E G A", w.Body.String())
}

func TestHandlers_CreateOrderWithModel_ReturnsSuccessJson(t *testing.T) {
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

func TestHandlers_CreateOrderWithModel_ReturnsFailureJson(t *testing.T) {
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

func TestHandlers_WasSuccessJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	wasSuccess(context, gin.H { "chuck" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"{\"chuck\":\"rhodes\"}", w.Body.String())
}

func TestHandlers_WasFailureJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	wasFailure(context, gin.H { "wendy" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"wendy\":\"rhodes\"}", w.Body.String())

}

func TestHandlers_SendResponseJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	sendResponse(context, gin.H { "robert" : "axelrod" }, http.StatusBadGateway)

	assert.Equal(t, w.Code, http.StatusBadGateway)
	assert.Equal(t,"{\"robert\":\"axelrod\"}", w.Body.String())
}

func TestHandlers_WasSuccessXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	wasSuccess(context, gin.H { "chuck" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"<map><chuck>rhodes</chuck></map>", w.Body.String())
}

func TestHandlers_WasFailureXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	wasFailure(context, gin.H { "wendy" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"<map><wendy>rhodes</wendy></map>", w.Body.String())

}

func TestHandlers_SendResponseXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	sendResponse(context, gin.H { "robert" : "axelrod" }, http.StatusBadGateway)

	assert.Equal(t, w.Code, http.StatusBadGateway)
	assert.Equal(t,"<map><robert>axelrod</robert></map>", w.Body.String())
}


// Helpers
func buildContextWithDefaultRouteRequest(w http.ResponseWriter, requestXml bool) *gin.Context {

	context, _ := gin.CreateTestContext(w)
	context.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	if requestXml {
		context.Request.Header.Add("Accept", "application/xml")
	}
	return context
}

func buildNewOrder() orders.Order  {
	return orders.NewOrder("market", "party", 0, 1,1, 1, 1234567890, 1)
}