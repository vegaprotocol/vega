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

package risk_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSwitchFromIsolatedMargin(t *testing.T) {
	e := getTestEngine(t, num.DecimalOne())
	evt := testMargin{
		party:       "party1",
		size:        1,
		price:       1000,
		asset:       "ETH",
		margin:      10,
		orderMargin: 20,
		general:     100000,
		market:      "ETH/DEC19",
	}
	// we're switching from margin to isolated, we expect to release all of the order margin
	e.as.EXPECT().InAuction().Return(false).AnyTimes()
	e.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	e.tsvc.EXPECT().GetTimeNow().AnyTimes()
	risk := e.SwitchFromIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne())
	require.Equal(t, num.NewUint(20), risk.Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(20), risk.Transfer().MinAmount)
	require.Equal(t, types.TransferTypeIsolatedMarginLow, risk.Transfer().Type)
	require.Equal(t, "party1", risk.Transfer().Owner)
	require.Equal(t, num.UintZero(), risk.MarginLevels().OrderMargin)
}

func TestSwithToIsolatedMarginContinuous(t *testing.T) {
	positionFactor := num.DecimalOne()
	e := getTestEngine(t, positionFactor)
	evt := testMargin{
		party:       "party1",
		size:        1,
		price:       1000,
		asset:       "ETH",
		margin:      10,
		orderMargin: 0,
		general:     500,
		market:      "ETH/DEC19",
	}
	// we're switching from margin to isolated, we expect to release all of the order margin
	e.as.EXPECT().InAuction().Return(false).AnyTimes()
	e.tsvc.EXPECT().GetTimeNow().AnyTimes()
	e.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	// margin factor too low - 0.01 * 1000 * 1 = 10 < 31 initial margin
	_, err := e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), []*types.Order{}, num.DecimalFromFloat(0.01), nil)
	require.Equal(t, "required position margin must be greater than initial margin", err.Error())

	// not enough in general account to cover
	// required position margin (600) + require order margin (0) - margin balance (10) > general account balance (500)
	_, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), []*types.Order{}, num.DecimalFromFloat(0.6), nil)
	require.Equal(t, "insufficient balance in general account to cover for required order margin", err.Error())

	// case1 - need to topup margin account only
	marginFactor := num.DecimalFromFloat(0.5)
	orders := []*types.Order{}
	riskEvent, err := e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, marginFactor, num.UintZero())
	require.NoError(t, err)
	require.Equal(t, 1, len(riskEvent))
	require.Equal(t, num.NewUint(490), riskEvent[0].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(490), riskEvent[0].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeMarginLow, riskEvent[0].Transfer().Type)
	require.Equal(t, "party1", riskEvent[0].Transfer().Owner)
	require.Equal(t, num.NewUint(0), riskEvent[0].MarginLevels().OrderMargin)

	buyOrderInfo, sellOrderInfo := extractOrderInfo(orders)
	requiredPositionMarginStatic, requiredOrderMarginStatic := risk.CalculateRequiredMarginInIsolatedMode(evt.size, evt.AverageEntryPrice().ToDecimal(), evt.Price().ToDecimal(), buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, nil)
	require.True(t, !requiredPositionMarginStatic.IsZero())
	require.True(t, requiredOrderMarginStatic.IsZero())
	transferRecalc := requiredPositionMarginStatic.Sub(evt.MarginBalance().ToDecimal())
	require.True(t, riskEvent[0].Transfer().Amount.Amount.ToDecimal().Sub(transferRecalc).IsZero())

	// case2 we have also some orders
	orders = []*types.Order{
		{Side: types.SideBuy, Remaining: 1, Price: num.NewUint(1000), Status: types.OrderStatusActive},
	}

	// not enough in general account to cover
	// required position margin (300) + require order margin (300) - margin balance (10) > general account balance (500)
	_, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, num.DecimalFromFloat(0.3), nil)
	require.Equal(t, "insufficient balance in general account to cover for required order margin", err.Error())

	evt.general = 10000
	marginFactor = num.DecimalFromFloat(0.3)
	riskEvent, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, marginFactor, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(riskEvent))
	require.Equal(t, num.NewUint(290), riskEvent[0].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(290), riskEvent[0].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeMarginLow, riskEvent[0].Transfer().Type)
	require.Equal(t, "party1", riskEvent[0].Transfer().Owner)
	require.Equal(t, num.NewUint(300), riskEvent[0].MarginLevels().OrderMargin)

	buyOrderInfo, sellOrderInfo = extractOrderInfo(orders)
	requiredPositionMarginStatic, requiredOrderMarginStatic = risk.CalculateRequiredMarginInIsolatedMode(evt.size, evt.AverageEntryPrice().ToDecimal(), evt.Price().ToDecimal(), buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, nil)
	require.True(t, !requiredPositionMarginStatic.IsZero())
	require.True(t, !requiredOrderMarginStatic.IsZero())
	transferRecalc = requiredPositionMarginStatic.Sub(evt.MarginBalance().ToDecimal())
	require.True(t, riskEvent[0].Transfer().Amount.Amount.ToDecimal().Sub(transferRecalc).IsZero())

	require.Equal(t, num.NewUint(300), riskEvent[1].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(300), riskEvent[1].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeOrderMarginLow, riskEvent[1].Transfer().Type)
	require.Equal(t, "party1", riskEvent[1].Transfer().Owner)
	require.Equal(t, num.NewUint(300), riskEvent[0].MarginLevels().OrderMargin)
	transferRecalc = requiredOrderMarginStatic.Sub(evt.OrderMarginBalance().ToDecimal())
	require.True(t, riskEvent[1].Transfer().Amount.Amount.ToDecimal().Sub(transferRecalc).IsZero())

	// case3 - need to release from margin account and order margin account back into general account
	evt.margin += 600
	evt.orderMargin += 400
	marginFactor = num.DecimalFromFloat(0.3)
	riskEvent, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, marginFactor, nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(riskEvent))
	require.Equal(t, num.NewUint(310), riskEvent[0].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(310), riskEvent[0].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeMarginHigh, riskEvent[0].Transfer().Type)
	require.Equal(t, "party1", riskEvent[0].Transfer().Owner)
	require.Equal(t, num.NewUint(300), riskEvent[0].MarginLevels().OrderMargin)

	buyOrderInfo, sellOrderInfo = extractOrderInfo(orders)
	requiredPositionMarginStatic, requiredOrderMarginStatic = risk.CalculateRequiredMarginInIsolatedMode(evt.size, evt.AverageEntryPrice().ToDecimal(), evt.Price().ToDecimal(), buyOrderInfo, sellOrderInfo, positionFactor, marginFactor, nil)
	require.True(t, !requiredPositionMarginStatic.IsZero())
	require.True(t, !requiredOrderMarginStatic.IsZero())
	transferRecalc = evt.MarginBalance().ToDecimal().Sub(requiredPositionMarginStatic)
	require.True(t, riskEvent[0].Transfer().Amount.Amount.ToDecimal().Sub(transferRecalc).IsZero())

	require.Equal(t, num.NewUint(100), riskEvent[1].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(100), riskEvent[1].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeOrderMarginHigh, riskEvent[1].Transfer().Type)
	require.Equal(t, "party1", riskEvent[1].Transfer().Owner)
	require.Equal(t, num.NewUint(300), riskEvent[0].MarginLevels().OrderMargin)

	transferRecalc = evt.OrderMarginBalance().ToDecimal().Sub(requiredOrderMarginStatic)
	require.True(t, riskEvent[1].Transfer().Amount.Amount.ToDecimal().Sub(transferRecalc).IsZero())
}

func extractOrderInfo(orders []*types.Order) (buyOrders, sellOrders []*risk.OrderInfo) {
	buyOrders, sellOrders = []*risk.OrderInfo{}, []*risk.OrderInfo{}
	for _, o := range orders {
		if o.Status == types.OrderStatusActive {
			remaining := o.TrueRemaining()
			price := o.Price.ToDecimal()
			isMarketOrder := o.Type == types.OrderTypeMarket
			if o.Side == types.SideBuy {
				buyOrders = append(buyOrders, &risk.OrderInfo{TrueRemaining: remaining, Price: price, IsMarketOrder: isMarketOrder})
			}
			if o.Side == types.SideSell {
				sellOrders = append(sellOrders, &risk.OrderInfo{TrueRemaining: remaining, Price: price, IsMarketOrder: isMarketOrder})
			}
		}
	}
	return buyOrders, sellOrders
}
