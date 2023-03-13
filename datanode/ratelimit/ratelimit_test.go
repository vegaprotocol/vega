package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/logging"
)

func TestRateLimit_HTTPMiddleware(t *testing.T) {
	count := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
	})
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/test", nil)

	cfg := NewDefaultConfig()
	r := NewFromConfig(&cfg, logging.NewTestLogger())

	limiter := r.HTTPMiddleware(handler)
	for i := 0; i < 100; i++ {
		res := httptest.NewRecorder()
		limiter.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, i+1, count)
	}

	for i := 0; i < 101; i++ {
		res := httptest.NewRecorder()
		limiter.ServeHTTP(res, req)
		require.Equal(t, http.StatusTooManyRequests, res.Code)
		assert.Equal(t, 100, count)
	}

	res := httptest.NewRecorder()
	limiter.ServeHTTP(res, req)
	require.Equal(t, http.StatusForbidden, res.Code)
	expiry := res.Header().Get("Retry-After")
	require.NotEmpty(t, expiry)
	assert.Equal(t, "600", expiry)
}
