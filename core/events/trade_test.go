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
