// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
	"github.com/stretchr/testify/assert"
)

func TestCachingValues(t *testing.T) {
	cache := matching.NewBookCache()

	// by default all cache are invalid
	_, priceOK := cache.GetIndicativePrice()
	assert.False(t, priceOK)
	_, volOK := cache.GetIndicativeVolume()
	assert.False(t, volOK)
	_, sideOK := cache.GetIndicativeUncrossingSide()
	assert.False(t, sideOK)

	// setting one value only validate one cache not the others
	cache.SetIndicativeVolume(42)
	_, priceOK = cache.GetIndicativePrice()
	assert.False(t, priceOK)
	vol, volOK := cache.GetIndicativeVolume()
	assert.True(t, volOK)
	assert.Equal(t, vol, uint64(42))
	_, sideOK = cache.GetIndicativeUncrossingSide()
	assert.False(t, sideOK)

	// setting all of them make them all valid
	cache.SetIndicativePrice(num.NewUint(84))
	cache.SetIndicativeUncrossingSide(types.SideBuy)
	price, priceOK := cache.GetIndicativePrice()
	assert.True(t, priceOK)
	assert.Equal(t, price.Uint64(), uint64(84))
	vol, volOK = cache.GetIndicativeVolume()
	assert.True(t, volOK)
	assert.Equal(t, vol, uint64(42))
	side, sideOK := cache.GetIndicativeUncrossingSide()
	assert.True(t, sideOK)
	assert.Equal(t, side, types.SideBuy)

	// invalide affects all cache
	cache.Invalidate()
	_, priceOK = cache.GetIndicativePrice()
	assert.False(t, priceOK)
	_, volOK = cache.GetIndicativeVolume()
	assert.False(t, volOK)
	_, sideOK = cache.GetIndicativeUncrossingSide()
	assert.False(t, sideOK)
}
