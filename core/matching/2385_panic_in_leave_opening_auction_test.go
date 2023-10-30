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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

func TestPanicInLeaveAuction(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "A",
			Side:        types.SideBuy,
			Price:       num.NewUint(100),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000001",
		},
		{
			MarketID:    market,
			Party:       "B",
			Side:        types.SideSell,
			Price:       num.NewUint(100),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000002",
		},
		{
			MarketID:    market,
			Party:       "C",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000003",
		},
		{
			MarketID:    market,
			Party:       "D",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000004",
		},
		{
			MarketID:    market,
			Party:       "E",
			Side:        types.SideSell,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000005",
		},
		{
			MarketID:    market,
			Party:       "F",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000006",
		},
		{
			MarketID:    market,
			Party:       "G",
			Side:        types.SideSell,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000007",
		},
		{
			MarketID:    market,
			Party:       "A",
			Side:        types.SideBuy,
			Price:       num.NewUint(120),
			Size:        33,
			Remaining:   33,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000008",
		},
	}

	// enter auction, should return no error and no orders
	cnlorders := book.EnterAuction()
	assert.Len(t, cnlorders, 0)

	for _, o := range orders {
		o := o
		o.OriginalPrice = o.Price.Clone()
		cnf, err := book.SubmitOrder(&o)
		assert.NoError(t, err)
		assert.Len(t, cnf.Trades, 0)
		assert.Len(t, cnf.PassiveOrdersAffected, 0)
	}

	cnf, porders, err := book.LeaveAuction(time.Now())
	assert.NoError(t, err)
	_ = cnf
	_ = porders
}
