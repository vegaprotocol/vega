// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package http_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	vghttp "code.vegaprotocol.io/vega/libs/http"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	ctx := context.Background()
	rl, err := vghttp.NewRateLimit(
		ctx,
		vghttp.RateLimitConfig{
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
	}
}
