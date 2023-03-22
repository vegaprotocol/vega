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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBookSimple_simpleLimitBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, uint64(1), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(1))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 1)
}

func TestOrderBookSimple_simpleLimitSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestOfferPriceAndVolume()
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, uint64(1), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 1)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(1))
	assert.Equal(t, len(book.ordersByID), 1)
}

func TestOrderBookSimple_simpleMarketBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleMarketSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

/*
 * NETWORK orders are the same as MARKET+FOK order so should not stay on the book
 * Make sure orders are cancelled and the book is left empty.
 */
func TestOrderBookSimple_simpleNetworkBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleNetworkSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

/*
 * Now we test simple orders against a book with orders in.
 */
func TestOrderBookSimple_simpleLimitBuyFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleLimitSellFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleMarketBuyFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeMarket,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleMarketSellFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeMarket,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleNetworkBuyFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleNetworkSellFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_FillAgainstGTTOrder(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeNetwork,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	price, volume, err := book.BestBidPriceAndVolume()
	assert.Error(t, err)
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}

func TestOrderBookSimple_simpleWashTrade(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, order2.Status, types.OrderStatusStopped)
}

func TestOrderBookSimple_simpleWashTradePartiallyFilledThenStopped(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order1 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000011",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, order2.Status, types.OrderStatusPartiallyFilled)
	assert.Equal(t, int(order2.Remaining), 1)
}

func TestOrderBookSimple_simpleWashTradePartiallyFilledThenStoppedDifferentPrices(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order1 := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(2),
		OriginalPrice: num.NewUint(2),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		ID:            "V0000000032-0000000011",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, order2.Status, types.OrderStatusPartiallyFilled)
	assert.Equal(t, int(order2.Remaining), 1)
}

type MarketPos struct {
	size, buy, sell               int64
	party                         string
	price                         *num.Uint
	buySumProduct, sellSumProduct *num.Uint
}

func (m MarketPos) Party() string {
	return m.party
}

func (m MarketPos) Size() int64 {
	return m.size
}

func (m MarketPos) Buy() int64 {
	return m.buy
}

func (m MarketPos) Sell() int64 {
	return m.sell
}

func (m MarketPos) Price() *num.Uint {
	if m.price != nil {
		return m.price
	}
	return num.UintZero()
}

func (m MarketPos) BuySumProduct() *num.Uint {
	if m.buySumProduct != nil {
		return m.buySumProduct
	}
	return num.UintZero()
}

func (m MarketPos) SellSumProduct() *num.Uint {
	if m.sellSumProduct != nil {
		return m.sellSumProduct
	}
	return num.UintZero()
}

func (m MarketPos) VWBuy() *num.Uint {
	if m.buySumProduct != nil && m.buy != 0 {
		return num.UintZero().Div(m.buySumProduct.Clone(), num.NewUint(uint64(m.buy)))
	}
	return num.UintZero()
}

func (m MarketPos) VWSell() *num.Uint {
	if m.sellSumProduct != nil && m.sell != 0 {
		return num.UintZero().Div(m.sellSumProduct.Clone(), num.NewUint(uint64(m.sell)))
	}
	return num.UintZero()
}

func TestOrderBookSimple_CancelDistressedOrders(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
		ID:            vgcrypto.RandomHash(),
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
		ID:            vgcrypto.RandomHash(),
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 1)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(10))
	assert.Equal(t, book.getTotalSellVolume(), uint64(10))
	assert.Equal(t, len(book.ordersByID), 2)

	// Now create a structure to contain the details of distressed party "A" and send them to be cancelled.
	parties := []events.MarketPosition{
		MarketPos{
			party: "A",
		},
	}
	orders, err := book.RemoveDistressedOrders(parties)
	require.NoError(t, err)
	assert.Equal(t, len(orders), 2)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.Equal(t, book.getTotalSellVolume(), uint64(0))
	assert.Equal(t, len(book.ordersByID), 0)
}
