package gin

import (
	"testing"
	"net/http/httptest"
	"net/http"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRoute(t *testing.T) {

	router := NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "V E G A", w.Body.String())
}