package rest

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	orderMocks "vega/api/trading/orders/mocks"
	tradeMocks "vega/api/trading/trades/mocks"
	"github.com/gin-gonic/gin"
)

func TestNewRouter_ExistsAndServesHttp(t *testing.T) {
	router := buildRouter()
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestRequestIdMiddleware(t *testing.T) {
	router := buildRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-Id"))
}

// Helpers
func buildRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	orderService := &orderMocks.MockOrderService{}
	tradeService := &tradeMocks.TradeService{}
	router := NewRouter(orderService, tradeService)
	return router
}