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
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type key struct {
	pos    int64
	status types.AMMPoolStatus
}

type priceVolume struct {
	price  *num.Uint
	volume uint64
}

// basically a cache of expensive calculation results given a pool's position/state
// we only keep one result each time and the map is just used as a quick lookup.
type poolCache struct {
	sell      map[key]*priceVolume // map pos+status -> best ask price and volume
	buy       map[key]*priceVolume // map pos+status -> best buy price and volume
	fairPrice map[int64]*num.Uint  // map pos -> fair-price
}

func NewPoolCache() *poolCache {
	return &poolCache{
		sell:      map[key]*priceVolume{},
		buy:       map[key]*priceVolume{},
		fairPrice: map[int64]*num.Uint{},
	}
}

func (pc *poolCache) getFairPrice(pos int64) (*num.Uint, bool) {
	if p, ok := pc.fairPrice[pos]; ok {
		return p.Clone(), true
	}
	return nil, false
}

func (pc *poolCache) setFairPrice(pos int64, fp *num.Uint) {
	pc.fairPrice = map[int64]*num.Uint{
		pos: fp.Clone(),
	}
}

func (pc *poolCache) getBestPrice(pos int64, side types.Side, status types.AMMPoolStatus) (*num.Uint, uint64, bool) {
	cache := pc.sell
	if side == types.SideBuy {
		cache = pc.buy
	}

	if pv, ok := cache[key{
		pos:    pos,
		status: status,
	}]; ok {
		return pv.price, pv.volume, true
	}
	return nil, 0, false
}

func (pc *poolCache) setBestPrice(pos int64, side types.Side, status types.AMMPoolStatus, price *num.Uint, volume uint64) {
	k := key{
		pos:    pos,
		status: status,
	}
	cache := map[key]*priceVolume{
		k: {
			price:  price.Clone(),
			volume: volume,
		},
	}

	if side == types.SideBuy {
		pc.buy = cache
		return
	}
	pc.sell = cache
}
