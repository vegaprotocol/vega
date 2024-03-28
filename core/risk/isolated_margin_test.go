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

package risk

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestCalcMarginForOrdersBySideBuyContinous(t *testing.T) {
	orders := []*types.Order{
		{Side: types.SideBuy, Remaining: 10, Price: num.NewUint(50), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 30, Price: num.NewUint(20), Status: types.OrderStatusActive},
	}
	currentPos := int64(0)
	marginFactor := num.DecimalFromFloat(0.5)
	positionFactor := num.DecimalFromInt64(10)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)

	// no position
	// orderMargin = 0.5*(10 * 50 + 20 * 40 + 30 * 20)/10 = 95
	orderSideMargin := calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(95), orderSideMargin)

	staticResult := CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position - similar to no position, nothing is covered
	// orderMargin = 0.5*(10 * 50 + 20 * 40 + 30 * 20)/10 = 95
	currentPos = 20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(95), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// part of the top order is covered, i.e. only 6 count:
	// orderMargin = 0.5*(6 * 50 + 20 * 40 + 30 * 20)/10 = 85
	currentPos = -4
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(85), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// all of the top order is covered, a some of the second one too
	// orderMargin = 0.5*(0 * 50 + 10 * 40 + 30 * 20)/10 = 50
	currentPos = -20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(50), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// all of the orders are covered by position on the other side
	currentPos = -60
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.UintZero(), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())
}

func TestCalcMarginForOrdersBySideSellContinous(t *testing.T) {
	orders := []*types.Order{
		{Side: types.SideSell, Remaining: 10, Price: num.NewUint(20), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 30, Price: num.NewUint(50), Status: types.OrderStatusActive},
	}
	currentPos := int64(0)
	marginFactor := num.DecimalFromFloat(0.5)
	positionFactor := num.DecimalFromInt64(10)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)

	// no position
	// orderMargin = 0.5*(10 * 20 + 20 * 40 + 30 * 50)/10 = 125
	orderSideMargin := calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(125), orderSideMargin)
	staticResult := CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position - similar to no position, nothing is covered
	// orderMargin = 0.5*(10 * 20 + 20 * 40 + 30 * 50)/10 = 125
	currentPos = -20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(125), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// part of the top order is covered, i.e. only 6 count:
	// orderMargin = 0.5*(6 * 20 + 20 * 40 + 30 * 50)/10 = 121
	currentPos = 4
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(121), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// all of the top order is covered, a some of the second one too
	// orderMargin = 0.5*(0 * 20 + 10 * 40 + 30 * 50)/10 = 95
	currentPos = 20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(95), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// all of the orders are covered by position on the other side
	currentPos = 60
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.UintZero(), calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, nil))
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())
}

func TestCalcMarginForOrdersBySideBuyAuction(t *testing.T) {
	orders := []*types.Order{
		{Side: types.SideBuy, Remaining: 10, Price: num.NewUint(50), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 30, Price: num.NewUint(20), Status: types.OrderStatusActive},
	}
	currentPos := int64(0)
	marginFactor := num.DecimalFromFloat(0.5)
	positionFactor := num.DecimalFromInt64(10)
	auctionPrice := num.NewUint(42)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)

	// no position
	// orderMargin = 0.5*(10 * 50 + 20 * 42 + 30 * 42)/10 = 130 (using the max between the order and auction price)
	orderSideMargin := calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(130), orderSideMargin)
	staticResult := CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position - similar to no position, nothing is covered (using the max between the order and auction price)
	// orderMargin = 0.5*(10 * 50 + 20 * 42 + 30 * 42)/10 = 130
	currentPos = 20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(130), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// part of the top order is covered, i.e. only 6 count:
	// orderMargin = 0.5*(6 * 50 + 20 * 42 + 30 * 42)/10 = 120
	currentPos = -4
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(120), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// all of the top order is covered, a some of the second one too
	// orderMargin = 0.5*(0 * 50 + 10 * 42 + 30 * 42)/10 = 84
	currentPos = -20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(84), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// all of the orders are covered by position on the other side
	currentPos = -60
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.UintZero(), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())
}

func TestCalcMarginForOrdersBySideSellAuction(t *testing.T) {
	orders := []*types.Order{
		{Side: types.SideSell, Remaining: 10, Price: num.NewUint(20), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 30, Price: num.NewUint(50), Status: types.OrderStatusActive},
	}
	currentPos := int64(0)
	marginFactor := num.DecimalFromFloat(0.5)
	positionFactor := num.DecimalFromInt64(10)
	auctionPrice := num.NewUint(42)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)

	// no position
	// orderMargin = 0.5*(10 * 42 + 20 * 42 + 30 * 50)/10 = 138
	orderSideMargin := calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(138), orderSideMargin)
	staticResult := CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position - similar to no position, nothing is covered
	// orderMargin = 0.5*(10 * 42 + 20 * 42 + 30 * 50)/10 = 138
	currentPos = -20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(138), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// part of the top order is covered, i.e. only 6 count:
	// orderMargin = 0.5*(6 * 42 + 20 * 42 + 30 * 50)/10 = 129
	currentPos = 4
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(129), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.RoundDown(0).String(), orderSideMargin.String()) // equal within rounding

	// long position
	// all of the top order is covered, a some of the second one too
	// orderMargin = 0.5*(0 * 42 + 10 * 42 + 30 * 50)/10 = 96
	currentPos = 20
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.NewUint(96), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// all of the orders are covered by position on the other side
	currentPos = 60
	orderSideMargin = calcOrderSideMargin(currentPos, orders, positionFactor, marginFactor, auctionPrice)
	require.Equal(t, num.UintZero(), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalFromUint(auctionPrice))
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())
}

func TestCalcOrderMarginContinous(t *testing.T) {
	orders := []*types.Order{
		{Side: types.SideSell, Remaining: 10, Price: num.NewUint(20), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 10, Price: num.NewUint(50), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 20, Price: num.NewUint(40), Status: types.OrderStatusActive},
		{Side: types.SideSell, Remaining: 30, Price: num.NewUint(50), Status: types.OrderStatusActive},
		{Side: types.SideBuy, Remaining: 30, Price: num.NewUint(20), Status: types.OrderStatusActive},
	}
	currentPos := int64(0)
	marginFactor := num.DecimalFromFloat(0.5)
	positionFactor := num.DecimalFromInt64(10)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)

	// no position
	// buy orderMargin = 0.5*(10 * 50 + 20 * 40 + 30 * 20)/10 = 95
	// sell orderMargin = 0.5*(10 * 20 + 20 * 40 + 30 * 50)/10 = 125
	// order margin = max(95,125) = 125
	orderSideMargin := CalcOrderMargins(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(125), orderSideMargin)
	staticResult := CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// long position
	// buy orderMargin = 0.5*(10 * 50 + 20 * 40 + 30 * 20)/10 = 95
	// sell orderMargin = 0.5*(6 * 20 + 20 * 40 + 30 * 50)/10 = 121
	currentPos = 4
	orderSideMargin = CalcOrderMargins(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(121), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// longer position
	// buy orderMargin = 0.5*(10 * 50 + 20 * 40 + 30 * 20)/10 = 95
	// sell orderMargin =  0.5*(0 * 20 + 5 * 40 + 30 * 50)/10 = 85
	currentPos = 25
	orderSideMargin = CalcOrderMargins(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(95), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// short position
	// buy orderMargin = 0.5*(6 * 50 + 20 * 40 + 30 * 20)/10 = 85
	// sell orderMargin = 0.5*(10 * 20 + 20 * 40 + 30 * 50)/10 = 125
	currentPos = -4
	orderSideMargin = CalcOrderMargins(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(125), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())

	// shorter position
	// buy orderMargin = 0.5*(0 * 50 + 10 * 40 + 30 * 20)/10 = 50
	// sell orderMargin = 0.5*(10 * 20 + 20 * 40 + 30 * 50)/10 = 125
	currentPos = -20
	orderSideMargin = CalcOrderMargins(currentPos, orders, positionFactor, marginFactor, nil)
	require.Equal(t, num.NewUint(125), orderSideMargin)
	staticResult = CalcOrderMarginIsolatedMode(currentPos, buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, num.DecimalZero())
	require.Equal(t, staticResult.Round(0).String(), orderSideMargin.String())
}

func TestGetIsolatedMarginTransfersOnPositionChangeIncrease(t *testing.T) {
	party := "Zohar"
	asset := "BTC"

	marginFactor := num.NewDecimalFromFloat(0.5)
	curMarginBalance := num.NewUint(1000)
	positionFactor := num.DecimalFromInt64(10)

	// go long trades
	trades := []*types.Trade{
		{Size: 5, Price: num.NewUint(12)},
		{Size: 10, Price: num.NewUint(10)},
	}

	// position going up from 0 to 15 (increasing)
	// required margin topup is equal to: 0.5 * (5*12+10*10)/10 = 8
	transfer := getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideBuy, 15, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)
	// i.e. take from order margin account to the margin account
	require.Equal(t, types.TransferTypeIsolatedMarginLow, transfer[0].Type)
	require.Equal(t, num.NewUint(8), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(8), transfer[0].MinAmount)

	// position going up from 0 to -15 (increasing)
	// go short trades
	trades = []*types.Trade{
		{Size: 10, Price: num.NewUint(10)},
		{Size: 5, Price: num.NewUint(12)},
	}
	// required margin topup is equal to: 0.5 * (5*12+10*10)/10 = 8
	transfer = getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideSell, -15, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)
	// i.e. take from order margin account to the margin account
	require.Equal(t, types.TransferTypeIsolatedMarginLow, transfer[0].Type)
	require.Equal(t, num.NewUint(8), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(8), transfer[0].MinAmount)
}

func TestGetIsolatedMarginTransfersOnPositionChangeDecrease(t *testing.T) {
	party := "Zohar"
	asset := "BTC"

	marginFactor := num.NewDecimalFromFloat(0.5)
	curMarginBalance := num.NewUint(40)
	positionFactor := num.DecimalFromInt64(10)

	trades := []*types.Trade{
		{Size: 5, Price: num.NewUint(12)},
		{Size: 10, Price: num.NewUint(10)},
	}
	markPrice := num.NewUint(12)
	// position going down from 20 to 5 (decreasing)
	// required margin topup is equal to: (40+20/10*-2)  * 15/20) = 27
	transfer := getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideSell, 5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), markPrice, false, false)
	// i.e. release from the margin account to the general account
	require.Equal(t, types.TransferTypeMarginHigh, transfer[0].Type)
	require.Equal(t, num.NewUint(27), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(27), transfer[0].MinAmount)

	// position going down from 20 to 5 (decreasing)
	trades = []*types.Trade{
		{Size: 5, Price: num.NewUint(10)},
		{Size: 10, Price: num.NewUint(12)},
	}
	// required margin release is equal to: (40+20/10*-1)  * 15/20) = 28
	transfer = getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideBuy, -5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), markPrice, false, false)
	// i.e. release from margin account general account
	require.Equal(t, types.TransferTypeMarginHigh, transfer[0].Type)
	require.Equal(t, num.NewUint(28), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(28), transfer[0].MinAmount)
}

func TestGetIsolatedMarginTransfersOnPositionChangeSwitchSides(t *testing.T) {
	party := "Zohar"
	asset := "BTC"

	marginFactor := num.NewDecimalFromFloat(0.5)
	curMarginBalance := num.NewUint(1000)
	positionFactor := num.DecimalFromInt64(10)

	trades := []*types.Trade{
		{Size: 15, Price: num.NewUint(11)},
		{Size: 10, Price: num.NewUint(12)},
	}
	// position going from 20 to -5 (switching sides)
	// required margin release is equal to: we release all 1000 margin, then require 0.5 * 5 * 12 / 10
	transfer := getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideSell, -5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)
	// i.e. release from the margin account to the general account
	require.Equal(t, types.TransferTypeMarginHigh, transfer[0].Type)
	require.Equal(t, num.NewUint(997), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(997), transfer[0].MinAmount)

	curMarginBalance = num.NewUint(1)
	transfer = getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideSell, -5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)

	// now we expect to need 2 more to be added from the order margin account
	require.Equal(t, types.TransferTypeMarginLow, transfer[0].Type)
	require.Equal(t, num.NewUint(2), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(2), transfer[0].MinAmount)

	curMarginBalance = num.NewUint(1000)
	trades = []*types.Trade{
		{Size: 10, Price: num.NewUint(12)},
		{Size: 15, Price: num.NewUint(11)},
	}
	// position going from -20 to 5 (switching sides)
	// required margin release is equal to: we release all 1000 margin, then require 0.5 * 5 * 11 / 10
	transfer = getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideBuy, 5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)
	// i.e. release from the margin account to the general account
	require.Equal(t, types.TransferTypeMarginHigh, transfer[0].Type)
	require.Equal(t, num.NewUint(998), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(998), transfer[0].MinAmount)

	// try the same as above for switching sides to short
	curMarginBalance = num.NewUint(1)
	transfer = getIsolatedMarginTransfersOnPositionChange(party, asset, trades, types.SideSell, -5, positionFactor, marginFactor, curMarginBalance, num.UintZero(), nil, false, false)

	// now we expect to need 1 more to be added from the order margin account
	require.Equal(t, types.TransferTypeMarginLow, transfer[0].Type)
	require.Equal(t, num.NewUint(1), transfer[0].Amount.Amount)
	require.Equal(t, num.NewUint(1), transfer[0].MinAmount)
}

func extractOrderInfo(orders []*types.Order) (buyOrders, sellOrders []*OrderInfo) {
	buyOrders, sellOrders = []*OrderInfo{}, []*OrderInfo{}
	for _, o := range orders {
		if o.Status == types.OrderStatusActive {
			remaining := o.TrueRemaining()
			price := o.Price.ToDecimal()
			isMarketOrder := o.Type == types.OrderTypeMarket
			if o.Side == types.SideBuy {
				buyOrders = append(buyOrders, &OrderInfo{TrueRemaining: remaining, Price: price, IsMarketOrder: isMarketOrder})
			}
			if o.Side == types.SideSell {
				sellOrders = append(sellOrders, &OrderInfo{TrueRemaining: remaining, Price: price, IsMarketOrder: isMarketOrder})
			}
		}
	}
	return buyOrders, sellOrders
}
