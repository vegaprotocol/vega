package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	for i := 0; i < cfg.Burst; i++ {
		res := httptest.NewRecorder()
		limiter.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, i+1, count)
	}

	for i := 0; i < cfg.Burst+1; i++ {
		res := httptest.NewRecorder()
		limiter.ServeHTTP(res, req)
		assert.Equal(t, http.StatusTooManyRequests, res.Code)
		assert.Equal(t, 100, count)
	}

	// We should have been banned after this so wait a second, then request again,
	// the ban time remaining should not be empty.
	time.Sleep(time.Second)

	res := httptest.NewRecorder()
	limiter.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
	expiry := res.Header().Get("RateLimit-Retry-After")
	assert.NotEmpty(t, expiry)
}
