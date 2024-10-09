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

package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestBidAndAskPresentAfterAuction(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_ = book.EnterAuction()

	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(1950),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	r := book.GetIndicativePriceAndVolume()
	assert.Equal(t, r.price.Uint64(), uint64(1975))
	assert.Equal(t, int(r.volume), 5)
	assert.Equal(t, r.side, types.SideBuy)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}

func TestBidAndAskPresentAfterAuctionInverse(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_ = book.EnterAuction()

	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(2050),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	r := book.GetIndicativePriceAndVolume()
	assert.Equal(t, r.price.Uint64(), uint64(1950))
	assert.Equal(t, int(r.volume), 5)
	assert.Equal(t, r.side, types.SideBuy)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}
