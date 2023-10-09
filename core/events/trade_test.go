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

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/assert"
)

func TestTradeDeepClone(t *testing.T) {
	ctx := context.Background()

	trade := &types.Trade{
		ID:          "Id",
		MarketID:    "MarketId",
		Price:       num.NewUint(1000),
		MarketPrice: num.NewUint(1000),
		Size:        2000,
		Buyer:       "Buyer",
		Seller:      "Seller",
		Aggressor:   proto.Side_SIDE_BUY,
		BuyOrder:    "BuyOrder",
		SellOrder:   "SellOrder",
		Timestamp:   3000,
		Type:        proto.Trade_TYPE_DEFAULT,
		BuyerFee: &types.Fee{
			MakerFee:          num.NewUint(4000),
			InfrastructureFee: num.NewUint(5000),
			LiquidityFee:      num.NewUint(6000),
		},
		SellerFee: &types.Fee{
			MakerFee:          num.NewUint(7000),
			InfrastructureFee: num.NewUint(8000),
			LiquidityFee:      num.NewUint(9000),
		},
		BuyerAuctionBatch:  10000,
		SellerAuctionBatch: 11000,
	}

	tEvent := events.NewTradeEvent(ctx, *trade)
	trade2 := tEvent.Trade()

	// Change the original values
	trade.ID = "Changed"
	trade.MarketID = "Changed"
	trade.Price = num.NewUint(999)
	trade.MarketPrice = num.NewUint(999)
	trade.Size = 999
	trade.Buyer = "Changed"
	trade.Seller = "Changed"
	trade.Aggressor = proto.Side_SIDE_UNSPECIFIED
	trade.BuyOrder = "Changed"
	trade.SellOrder = "Changed"
	trade.Timestamp = 999
	trade.Type = proto.Trade_TYPE_UNSPECIFIED
	trade.BuyerFee.MakerFee = num.NewUint(999)
	trade.BuyerFee.InfrastructureFee = num.NewUint(999)
	trade.BuyerFee.LiquidityFee = num.NewUint(999)
	trade.SellerFee.MakerFee = num.NewUint(999)
	trade.SellerFee.InfrastructureFee = num.NewUint(999)
	trade.SellerFee.LiquidityFee = num.NewUint(999)
	trade.BuyerAuctionBatch = 999
	trade.SellerAuctionBatch = 999

	// Check things have changed
	assert.NotEqual(t, trade.ID, trade2.Id)
	assert.NotEqual(t, trade.MarketID, trade2.MarketId)
	assert.NotEqual(t, trade.Price, trade2.Price)
	assert.NotEqual(t, trade.Size, trade2.Size)
	assert.NotEqual(t, trade.Buyer, trade2.Buyer)
	assert.NotEqual(t, trade.Seller, trade2.Seller)
	assert.NotEqual(t, trade.Aggressor, trade2.Aggressor)
	assert.NotEqual(t, trade.BuyOrder, trade2.BuyOrder)
	assert.NotEqual(t, trade.SellOrder, trade2.SellOrder)
	assert.NotEqual(t, trade.Timestamp, trade2.Timestamp)
	assert.NotEqual(t, trade.Type, trade2.Type)
	assert.NotEqual(t, trade.BuyerFee.MakerFee, trade2.BuyerFee.MakerFee)
	assert.NotEqual(t, trade.BuyerFee.InfrastructureFee, trade2.BuyerFee.InfrastructureFee)
	assert.NotEqual(t, trade.BuyerFee.LiquidityFee, trade2.BuyerFee.LiquidityFee)
	assert.NotEqual(t, trade.SellerFee.MakerFee, trade2.SellerFee.MakerFee)
	assert.NotEqual(t, trade.SellerFee.InfrastructureFee, trade2.SellerFee.InfrastructureFee)
	assert.NotEqual(t, trade.SellerFee.LiquidityFee, trade2.SellerFee.LiquidityFee)
	assert.NotEqual(t, trade.BuyerAuctionBatch, trade2.BuyerAuctionBatch)
	assert.NotEqual(t, trade.SellerAuctionBatch, trade2.SellerAuctionBatch)
}
