package gin

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
)

type MockOrderService struct {}

func (p *MockOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error) {
	return true, nil
}

func TestNewRouter_ExistsAndServesHttp(t *testing.T) {

	orderService := &MockOrderService{}
	router := NewRouter(orderService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}