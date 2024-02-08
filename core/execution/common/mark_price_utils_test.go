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

package common_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/require"
)

func TestMarkPriceByWeights(t *testing.T) {
	// no mark prices
	require.Nil(t, common.CompositePriceByWeight(nil, []num.Decimal{num.NewDecimalFromFloat(0.2), num.NewDecimalFromFloat(0.5)}, []int64{3, 2}, []time.Duration{0, 0}, 10))

	// no non-stale mark prices
	// time now is 10, both timestamps of prices are 2,3 with delta = 0 for both
	require.Nil(t, common.CompositePriceByWeight([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []num.Decimal{num.NewDecimalFromFloat(0.2), num.NewDecimalFromFloat(0.5)}, []int64{3, 2}, []time.Duration{0, 0}, 10))

	// only the first price is non stale
	// 10-8<=2
	require.Equal(t, num.NewUint(100), common.CompositePriceByWeight([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []num.Decimal{num.NewDecimalFromFloat(0.2), num.NewDecimalFromFloat(0.5)}, []int64{8, 2}, []time.Duration{2, 0}, 10))

	// only the second price is non stale
	// 10-7<=3
	require.Equal(t, num.NewUint(80), common.CompositePriceByWeight([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []num.Decimal{num.NewDecimalFromFloat(0.2), num.NewDecimalFromFloat(0.5)}, []int64{8, 7}, []time.Duration{1, 3}, 10))

	// both prices are eligible use weights:
	// 0.4*100+0.6*80 = 88
	require.Equal(t, num.NewUint(88), common.CompositePriceByWeight([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []num.Decimal{num.NewDecimalFromFloat(0.2), num.NewDecimalFromFloat(0.3)}, []int64{8, 7}, []time.Duration{2, 3}, 10))
}

func TestMarkPriceByMedian(t *testing.T) {
	// no prices
	require.Nil(t, common.CompositePriceByMedian(nil, []int64{3, 2}, []time.Duration{0, 0}, 10))

	// no non-stale mark prices
	// time now is 10, both timestamps of prices are 2,3 with delta = 0 for both
	require.Nil(t, common.CompositePriceByMedian([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []int64{3, 2}, []time.Duration{0, 0}, 10))

	// only the first price is non stale
	// 10-8<=2
	require.Equal(t, num.NewUint(100), common.CompositePriceByMedian([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []int64{8, 2}, []time.Duration{2, 0}, 10))

	// only the second price is non stale
	// 10-7<=3
	require.Equal(t, num.NewUint(80), common.CompositePriceByMedian([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []int64{8, 7}, []time.Duration{1, 3}, 10))

	// both prices are non stale, median is calculated (average in this even case)
	require.Equal(t, num.NewUint(90), common.CompositePriceByMedian([]*num.Uint{num.NewUint(100), num.NewUint(80)}, []int64{8, 7}, []time.Duration{2, 3}, 10))

	// all prices are non stale, median is calculated
	require.Equal(t, num.NewUint(99), common.CompositePriceByMedian([]*num.Uint{num.NewUint(99), num.NewUint(100), num.NewUint(80)}, []int64{8, 8, 7}, []time.Duration{2, 2, 3}, 10))
}

func TestMedianMarkPrice(t *testing.T) {
	require.Nil(t, common.MedianPrice(nil))
	require.Equal(t, "100", common.MedianPrice([]*num.Uint{num.NewUint(110), num.NewUint(99), num.NewUint(100)}).String())
	require.Equal(t, "100", common.MedianPrice([]*num.Uint{num.NewUint(110), num.NewUint(101), num.NewUint(99), num.NewUint(100)}).String())
}

func TestMarkPriceFromTrades(t *testing.T) {
	alpha := num.DecimalZero()
	decayPower := num.DecimalZero()
	lambda := num.NewDecimalFromFloat(100)

	trade1 := &types.Trade{
		Price:     num.NewUint(129),
		Size:      10,
		Timestamp: 120,
	}

	trade2 := &types.Trade{
		Price:     num.NewUint(124),
		Size:      40,
		Timestamp: 150,
	}
	trade3 := &types.Trade{
		Price:     num.NewUint(133),
		Size:      50,
		Timestamp: 200,
	}

	// given alpha is 0, the time_weight is 1
	// the total size is 60, so trade weights are:
	// 1/10, 4/10, 5/10
	// so the markPrice = 0.1*129 + 0.4*124 + 0.5 * 133 = 129
	mp := common.PriceFromTrades([]*types.Trade{trade1, trade2, trade3}, alpha, lambda, decayPower, 200)
	require.Equal(t, "129", mp.String())

	// now lets repeat with non zero alpha
	alpha = num.DecimalFromFloat(0.2)
	decayPower = num.DecimalOne()

	// given alpha is 0, the time_weight is 1
	// the total size is 60, so trade weights are:
	// 1/10 * (1 - 0.2 * (200-120)/100) = 0.084
	// 4/10 * (1 - 0.2 * (200-150)/100) = 0.36
	// 5/10 * (1 - 0.2 * (200-200)/100) = 0.5
	// total weight = 0.944
	// mp = (0.084 * 129 + 0.36 * 124 + 0.5 * 133)/0.944 = 172.1276595745
	mp = common.PriceFromTrades([]*types.Trade{trade1, trade2, trade3}, alpha, lambda, decayPower, 200)
	require.Equal(t, "129", mp.String())
}

func TestPBookAtTimeT(t *testing.T) {
	book := matching.NewCachedOrderBook(logging.NewTestLogger(), matching.NewDefaultConfig(), "market1", false, func(int64) {})
	C := num.NewUint(1000)
	initialScaling := num.DecimalFromFloat(0.2)
	slippage := num.DecimalFromFloat(0.1)
	shortRisk := num.DecimalFromFloat(0.3)
	longRisk := num.DecimalFromFloat(0.4)

	// empty book
	require.Nil(t, common.PriceFromBookAtTime(C, initialScaling, slippage, shortRisk, longRisk, book))

	// no bids
	_, err := book.SubmitOrder(newOrder(num.NewUint(120), 10, types.SideSell))
	require.NoError(t, err)
	require.Nil(t, common.PriceFromBookAtTime(C, initialScaling, slippage, shortRisk, longRisk, book))
	book.CancelAllOrders("party1")

	// no asks
	_, err = book.SubmitOrder(newOrder(num.NewUint(125), 10, types.SideBuy))
	require.NoError(t, err)
	require.Nil(t, common.PriceFromBookAtTime(C, initialScaling, slippage, shortRisk, longRisk, book))

	// orders on both sides
	_, err = book.SubmitOrder(newOrder(num.NewUint(200), 10, types.SideSell))
	require.NoError(t, err)

	// N_buy = 1000 / ((0.2) * (0.1+0.3)) = 12500
	// N_sell = 1000 / ((0.2) * (0.1+0.4)) = 10000
	// V_buy = N_buy/best_bid = 12500/125 = 100
	// V_sell = N_sell/best_ask = 10000/200 = 50
	// insufficient volume in the book for both sides

	require.Nil(t, common.PriceFromBookAtTime(C, initialScaling, slippage, shortRisk, longRisk, book))

	// add orders on both sides
	_, err = book.SubmitOrder(newOrder(num.NewUint(200), 40, types.SideSell))
	require.NoError(t, err)
	_, err = book.SubmitOrder(newOrder(num.NewUint(125), 90, types.SideBuy))
	require.NoError(t, err)

	// (125+200)/2 = 162
	require.Equal(t, "162", common.PriceFromBookAtTime(C, initialScaling, slippage, shortRisk, longRisk, book).String())
}

func TestCalculateTimeWeightedAverageBookMarkPrice(t *testing.T) {
	timeToPrice := map[int64]*num.Uint{0: num.NewUint(100), 30: num.NewUint(120), 45: num.NewUint(150)}

	// 100 * 30/60 + 120 * 15/60 + 150 * 15/60 = 117.5 => 117
	require.Equal(t, "117", common.CalculateTimeWeightedAverageBookPrice(timeToPrice, 60, 60).String())

	// 120 * 15/30 + 150 * 15/30 = 97.5 => 135
	require.Equal(t, "135", common.CalculateTimeWeightedAverageBookPrice(timeToPrice, 60, 30).String())

	// 100 * 30/120 + 120 * 15/120 + 150 * 75/120 = 133.75 => 133
	require.Equal(t, "133", common.CalculateTimeWeightedAverageBookPrice(timeToPrice, 120, 120).String())

	// only the price from 45 is considered as the price from 30 is starting before the mark price period
	require.Equal(t, "150", common.CalculateTimeWeightedAverageBookPrice(timeToPrice, 120, 80).String())
}

func newOrder(price *num.Uint, size uint64, side types.Side) *types.Order {
	return &types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      "market1",
		Party:         "party1",
		Side:          side,
		Price:         price,
		OriginalPrice: price,
		Size:          size,
		Remaining:     size,
		TimeInForce:   types.OrderTimeInForceGTC,
	}
}
