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

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, indicativePrice.Uint64(), uint64(1975))
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.SideBuy)
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

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, indicativePrice.Uint64(), uint64(1950))
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.SideBuy)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}
