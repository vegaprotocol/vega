package gin

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
)

func TestIndexRoute_MappedCorrectly(t *testing.T) {
	handlers := Handlers {}
	r := handlers.IndexRoute()
	assert.Equal(t, "/", r)
}

func TestCreateOrderRoute_MappedCorrectly(t *testing.T) {
	handlers := Handlers {}
	r := handlers.CreateOrderRoute()
	assert.Equal(t, "/orders/create", r)
}

func TestIndexHandler_ReturnsExpectedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	handlers := Handlers {}
	handlers.Index(context)

	context.Request, _ = http.NewRequest(http.MethodGet, handlers.IndexRoute(), nil)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "V E G A", w.Body.String())
}

func TestCreateOrderHandler_ReturnsExpectedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	orderService := &MockOrderService{}
	handlers := Handlers {
		OrderService: orderService,
	}
	handlers.CreateOrder(context)

	context.Request, _ = http.NewRequest(http.MethodGet, handlers.CreateOrderRoute(), nil)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, "null", w.Body.String())
}