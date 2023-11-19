// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
