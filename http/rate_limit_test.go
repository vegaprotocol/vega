package http_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	vhttp "code.vegaprotocol.io/vega/http"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	ctx := context.Background()
	rl, err := vhttp.NewRateLimit(
		ctx,
		vhttp.RateLimitConfig{
			CoolDown:  encoding.Duration{Duration: 1 * time.Minute},
			AllowList: []string{"1.2.3.4/32", "2.3.4.252/30", "fe80::/10"},
		},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, rl) {
		// IP addresses in the allow list
		for _, ip := range []string{"1.2.3.4", "2.3.4.254", "fe80::abcd"} {
			for i := 0; i < 10; i++ {
				err = rl.NewRequest("someprefix", ip)
				assert.NoError(t, err)
			}
		}

		// IP address not in the allow list
		err = rl.NewRequest("someprefix", "2.2.2.2")
		assert.NoError(t, err)
		err = rl.NewRequest("someprefix", "2.2.2.2")
		assert.Error(t, err)

		err = rl.NewRequest("someprefix", "1a2b::abcd")
		assert.NoError(t, err)
		err = rl.NewRequest("someprefix", "1a2b::abcd")
		assert.Error(t, err)
	}
}
