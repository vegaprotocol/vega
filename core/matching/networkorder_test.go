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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

// reproducing bug from https://github.com/vegaprotocol/vega/issues/2180

func TestNetworkOrder_ValidAveragedPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []types.Order{
		{
			MarketID:      market,
			Status:        types.OrderStatusActive,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          4,
			Remaining:     4,
			TimeInForce:   types.OrderTimeInForceGTC,
			Type:          types.OrderTypeLimit,
			ID:            "v0000000000000-0000001",
		},
		{
			MarketID:      market,
			Status:        types.OrderStatusActive,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(75),
			OriginalPrice: num.NewUint(75),
			Size:          4,
			Remaining:     4,
			TimeInForce:   types.OrderTimeInForceGTC,
			Type:          types.OrderTypeLimit,
			ID:            "v0000000000000-0000002",
		},
	}

	var (
		totalSize                 uint64
		totalPrice, expectedPrice = num.UintZero(), num.NewUint(0)
	)
	for _, v := range orders {
		v := v
		_, err := book.ob.SubmitOrder(&v)
		assert.NoError(t, err)
		// totalPrice += v.Price * v.Size
		totalPrice.Add(
			totalPrice,
			num.UintZero().Mul(v.Price, num.NewUint(v.Size)),
		)
		totalSize += v.Size
	}
	expectedPrice.Div(totalPrice, num.NewUint(totalSize))
	assert.Equal(t, uint64(87), expectedPrice.Uint64())

	// now let's place the network order and validate it's price
	netorder := types.Order{
		MarketID:      market,
		Size:          8,
		Remaining:     8,
		Status:        types.OrderStatusActive,
		Party:         "network",
		Side:          types.SideSell,
		Price:         num.UintZero(),
		OriginalPrice: num.UintZero(),
		CreatedAt:     0,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}

	_, err := book.ob.SubmitOrder(&netorder)
	assert.NoError(t, err)
	// now we expect the price of the order to be updated
	assert.Equal(t, expectedPrice.Uint64(), netorder.Price.Uint64())
}
