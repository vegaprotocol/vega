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

package entities_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestProtoFromTrade(t *testing.T) {
	vegaTime := time.Now()
	priceString := "1000035452"
	price, _ := decimal.NewFromString(priceString)

	idString := "BC2001BDDAC588F8AAAE0D9BEC3D6881A447B888447E5D0A9DE92D149BA4E877"
	marketIDString := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	size := uint64(5)
	buyerIDString := "2e4f34a38204a2a155be678e670903ed8df96e813700729deacd3daf7e55039e"
	sellerIDString := "8b6be1a03cc4d529f682887a78b66e6879d17f81e2b37356ca0acbc5d5886eb8"
	buyOrderIDString := "CF951606211775C43449807FE15F908704A85C514D65D549D67BBD6B5EEF66BB"
	sellOrderIDString := "6A94947F724CDB7851BEE793ACA6888F68ABBF8D49DFD0F778424A7CE42E7B7D"

	trade := entities.Trade{
		VegaTime:                vegaTime,
		ID:                      entities.TradeID(idString),
		MarketID:                entities.MarketID(marketIDString),
		Price:                   price,
		Size:                    size,
		Buyer:                   entities.PartyID(buyerIDString),
		Seller:                  entities.PartyID(sellerIDString),
		Aggressor:               entities.SideBuy,
		BuyOrder:                entities.OrderID(buyOrderIDString),
		SellOrder:               entities.OrderID(sellOrderIDString),
		Type:                    entities.TradeTypeNetworkCloseOutGood,
		BuyerMakerFee:           decimal.NewFromInt(2),
		BuyerInfrastructureFee:  decimal.NewFromInt(3),
		BuyerLiquidityFee:       decimal.NewFromInt(4),
		SellerMakerFee:          decimal.NewFromInt(1),
		SellerInfrastructureFee: decimal.NewFromInt(10),
		SellerLiquidityFee:      decimal.NewFromInt(100),
		BuyerAuctionBatch:       3,
		SellerAuctionBatch:      4,
	}

	p := trade.ToProto()

	assert.Equal(t, vegaTime.UnixNano(), p.Timestamp)
	assert.Equal(t, idString, p.Id)
	assert.Equal(t, marketIDString, p.MarketId)
	assert.Equal(t, priceString, p.Price)
	assert.Equal(t, size, p.Size)
	assert.Equal(t, buyerIDString, p.Buyer)
	assert.Equal(t, sellerIDString, p.Seller)
	assert.Equal(t, vega.Side_SIDE_BUY, p.Aggressor)
	assert.Equal(t, buyOrderIDString, p.BuyOrder)
	assert.Equal(t, sellOrderIDString, p.SellOrder)
	assert.Equal(t, vega.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD, p.Type)
	assert.Equal(t, "2", p.BuyerFee.MakerFee)
	assert.Equal(t, "3", p.BuyerFee.InfrastructureFee)
	assert.Equal(t, "4", p.BuyerFee.LiquidityFee)
	assert.Equal(t, "1", p.SellerFee.MakerFee)
	assert.Equal(t, "10", p.SellerFee.InfrastructureFee)
	assert.Equal(t, "100", p.SellerFee.LiquidityFee)
	assert.Equal(t, uint64(3), p.BuyerAuctionBatch)
	assert.Equal(t, uint64(4), p.SellerAuctionBatch)
}

func TestTradeFromProto(t *testing.T) {
	tradeEventProto := vega.Trade{
		Id:        "521127F24B1FA40311BA2FB3F6977310346346604B275DB7B767B04240A5A5C3",
		MarketId:  "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8",
		Price:     "1000097674",
		Size:      1,
		Buyer:     "b4376d805a888548baabfae74ef6f4fa4680dc9718bab355fa7191715de4fafe",
		Seller:    "539e8c7c8c15044a6b37a8bf4d7d988588b2f63ed48666b342bc530c8312e002",
		Aggressor: vega.Side_SIDE_SELL,
		BuyOrder:  "0976E6CFE1513C46D5EC8877EFB51E6F12EB24709131D08EF358310FA4409158",
		SellOrder: "459B8150105322406C1CEABF596E0E13ED113A98C1E290E2144D7A6236EDC6C2",
		Timestamp: 1644573750767832307,
		Type:      vega.Trade_TYPE_DEFAULT,
		BuyerFee: &vega.Fee{
			MakerFee:          "4000142",
			InfrastructureFee: "10000036",
			LiquidityFee:      "10000355",
		},
		SellerFee:          nil,
		BuyerAuctionBatch:  3,
		SellerAuctionBatch: 0,
	}

	testVegaTime := time.Now()
	trade, err := entities.TradeFromProto(&tradeEventProto, generateTxHash(), testVegaTime, 5)
	if err != nil {
		t.Fatalf("failed to convert proto to trade:%s", err)
	}

	assert.Equal(t, testVegaTime.Add(5*time.Microsecond), trade.SyntheticTime)
	assert.Equal(t, testVegaTime, trade.VegaTime)
	assert.Equal(t, uint64(5), trade.SeqNum)

	assert.Equal(t, tradeEventProto.Id, trade.ID.String())
	assert.Equal(t, tradeEventProto.MarketId, trade.MarketID.String())
	price, _ := decimal.NewFromString(tradeEventProto.Price)
	assert.Equal(t, price, trade.Price)
	size := tradeEventProto.Size
	assert.Equal(t, size, trade.Size)
	assert.Equal(t, tradeEventProto.Buyer, trade.Buyer.String())
	assert.Equal(t, tradeEventProto.Seller, trade.Seller.String())
	assert.Equal(t, entities.SideSell, trade.Aggressor)
	assert.Equal(t, tradeEventProto.BuyOrder, trade.BuyOrder.String())
	assert.Equal(t, tradeEventProto.SellOrder, trade.SellOrder.String())
	assert.Equal(t, entities.TradeTypeDefault, trade.Type)

	buyerMakerFee, _ := decimal.NewFromString(tradeEventProto.BuyerFee.MakerFee)
	buyerLiquidityFee, _ := decimal.NewFromString(tradeEventProto.BuyerFee.LiquidityFee)
	buyerInfraFee, _ := decimal.NewFromString(tradeEventProto.BuyerFee.InfrastructureFee)
	assert.Equal(t, buyerMakerFee, trade.BuyerMakerFee)
	assert.Equal(t, buyerLiquidityFee, trade.BuyerLiquidityFee)
	assert.Equal(t, buyerInfraFee, trade.BuyerInfrastructureFee)
	assert.Equal(t, decimal.Zero, trade.SellerMakerFee)
	assert.Equal(t, decimal.Zero, trade.SellerLiquidityFee)
	assert.Equal(t, decimal.Zero, trade.SellerInfrastructureFee)
	assert.Equal(t, tradeEventProto.BuyerAuctionBatch, trade.BuyerAuctionBatch)
	assert.Equal(t, tradeEventProto.SellerAuctionBatch, trade.SellerAuctionBatch)
}
