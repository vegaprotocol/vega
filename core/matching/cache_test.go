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

package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
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
