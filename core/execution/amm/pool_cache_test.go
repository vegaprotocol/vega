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

package amm

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestAMMCache(t *testing.T) {
	t.Run("test fair-price cache", testFairPriceCache)
	t.Run("test best-price cache", testBestPriceCache)
}

func testFairPriceCache(t *testing.T) {
	c := NewPoolCache()

	p, ok := c.getFairPrice(0)
	assert.False(t, ok)
	assert.Nil(t, p)

	// add something
	c.setFairPrice(100, num.NewUint(123))
	p, ok = c.getFairPrice(100)
	assert.True(t, ok)
	assert.Equal(t, "123", p.String())

	// replace it
	c.setFairPrice(1001, num.NewUint(321))
	p, ok = c.getFairPrice(100)
	assert.False(t, ok)
	assert.Nil(t, p)

	// get new value
	p, ok = c.getFairPrice(1001)
	assert.True(t, ok)
	assert.Equal(t, "321", p.String())
}

func testBestPriceCache(t *testing.T) {
	c := NewPoolCache()

	p, v, ok := c.getBestPrice(0, types.SideBuy, types.AMMPoolStatusActive)
	assert.False(t, ok)
	assert.Nil(t, p)
	assert.Zero(t, v)

	// add something to buy cache
	c.setBestPrice(100, types.SideBuy, types.AMMPoolStatusActive, num.NewUint(123), 321)

	// now get it back
	p, v, ok = c.getBestPrice(100, types.SideBuy, types.AMMPoolStatusActive)
	assert.True(t, ok)
	assert.Equal(t, "123", p.String())
	assert.Equal(t, 321, int(v))

	// now try to get the other side
	p, v, ok = c.getBestPrice(100, types.SideSell, types.AMMPoolStatusActive)
	assert.False(t, ok)
	assert.Nil(t, p)
	assert.Zero(t, v)

	// now try to get it with a different status
	p, v, ok = c.getBestPrice(100, types.SideBuy, types.AMMPoolStatusReduceOnly)
	assert.False(t, ok)
	assert.Nil(t, p)
	assert.Zero(t, v)

	// now add one for the other side
	c.setBestPrice(100, types.SideSell, types.AMMPoolStatusActive, num.NewUint(12300), 32100)

	// check we can still get the buy one
	p, v, ok = c.getBestPrice(100, types.SideBuy, types.AMMPoolStatusActive)
	assert.True(t, ok)
	assert.Equal(t, "123", p.String())
	assert.Equal(t, 321, int(v))

	// and also the sell one
	p, v, ok = c.getBestPrice(100, types.SideSell, types.AMMPoolStatusActive)
	assert.True(t, ok)
	assert.Equal(t, "12300", p.String())
	assert.Equal(t, 32100, int(v))
}
