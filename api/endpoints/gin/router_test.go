package gin

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
)

type MockTradingService struct {}

func (p *MockTradingService) CreateOrder(s string) string {
	return "SUCCESS"
}

func TestNewRouter_ExistsAndServesHttp(t *testing.T) {

	tradingService := &MockTradingService{}
	router := NewRouter(tradingService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}