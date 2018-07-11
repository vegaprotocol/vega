package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vega/api/mocks"

	"github.com/stretchr/testify/assert"
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
	orderService := &mocks.OrderService{}
	tradeService := &mocks.TradeService{}
	router := NewRouter(orderService, tradeService)
	return router
}
