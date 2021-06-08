package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	cache.SetIndicativeUncrossingSide(types.Side_SIDE_BUY)
	price, priceOK := cache.GetIndicativePrice()
	assert.True(t, priceOK)
	assert.Equal(t, price.Uint64(), uint64(84))
	vol, volOK = cache.GetIndicativeVolume()
	assert.True(t, volOK)
	assert.Equal(t, vol, uint64(42))
	side, sideOK := cache.GetIndicativeUncrossingSide()
	assert.True(t, sideOK)
	assert.Equal(t, side, types.Side_SIDE_BUY)

	// invalide affects all cache
	cache.Invalidate()
	_, priceOK = cache.GetIndicativePrice()
	assert.False(t, priceOK)
	_, volOK = cache.GetIndicativeVolume()
	assert.False(t, volOK)
	_, sideOK = cache.GetIndicativeUncrossingSide()
	assert.False(t, sideOK)

}
