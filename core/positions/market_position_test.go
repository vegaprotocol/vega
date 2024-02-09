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

package positions_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestUpdateInPlace(t *testing.T) {
	pos := positions.NewMarketPosition("zohar")
	buy := &types.Order{
		ID:        "1",
		MarketID:  "2",
		Party:     "zohar",
		Side:      types.SideBuy,
		Size:      10,
		Price:     num.NewUint(100),
		Remaining: 10,
	}
	sell := &types.Order{
		ID:        "2",
		MarketID:  "2",
		Party:     "zohar",
		Side:      types.SideSell,
		Size:      20,
		Price:     num.NewUint(200),
		Remaining: 20,
	}
	pos.RegisterOrder(nil, buy)
	pos.RegisterOrder(nil, sell)
	trade1 := &types.Trade{
		ID:    "t1",
		Size:  3,
		Price: num.NewUint(120),
	}
	updatedPos := pos.UpdateInPlaceOnTrades(nil, types.SideBuy, []*types.Trade{trade1}, buy)
	require.Equal(t, int64(3), updatedPos.Size())
	require.Equal(t, int64(7), updatedPos.Buy())
	require.Equal(t, int64(20), updatedPos.Sell())

	// now trade the whole size of the sell order
	trade2 := &types.Trade{
		ID:    "t2",
		Size:  20,
		Price: num.NewUint(150),
	}
	updatedPos = updatedPos.UpdateInPlaceOnTrades(nil, types.SideSell, []*types.Trade{trade2}, sell)
	require.Equal(t, int64(-17), updatedPos.Size())
	require.Equal(t, int64(7), updatedPos.Buy())
	require.Equal(t, int64(0), updatedPos.Sell())

	// now unregister the remaining buy order
	buy.Remaining = 7

	updatedPos.UnregisterOrder(nil, buy)
	require.Equal(t, int64(-17), updatedPos.Size())
	require.Equal(t, int64(0), updatedPos.Buy())
	require.Equal(t, int64(0), updatedPos.Sell())
}
