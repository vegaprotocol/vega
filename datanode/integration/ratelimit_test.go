package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	url := "http://localhost:3008/api/v2/info"
	for {
		// keep making requests against the http API until we get a response that is not 200
		// this response should be 429
		resp, err := http.Get(url)
		require.NoError(t, err)
		if resp.StatusCode != http.StatusOK {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			break
		}
		assert.Len(t, resp.Header.Get("RateLimit-Retry-After"), 0)
		_ = resp.Body.Close()
	}

	for {
		// continue making requests against the http API until we get a response that is not 429 or 200
		// this response should be 403 (Forbidden)
		// we have to check for 200 because in the time it takes to make the requests and get a response
		// our token bucket may have refilled allowing us to make more requests
		resp, err := http.Get(url)
		require.NoError(t, err)
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusOK {
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Len(t, resp.Header.Get("RateLimit-Retry-After"), 3)
			break
		}
		assert.Len(t, resp.Header.Get("RateLimit-Retry-After"), 0)
		_ = resp.Body.Close()
	}
}
