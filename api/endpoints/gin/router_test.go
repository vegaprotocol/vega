package gin

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	"vega/api/mocks"
)


func TestNewRouter_ExistsAndServesHttp(t *testing.T) {

	orderService := &mocks.MockOrderService{}
	router := NewRouter(orderService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}