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
	e.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Return(num.NewUint(999), nil)
	risk := e.SwitchFromIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne())
	require.Equal(t, num.NewUint(20), risk.Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(20), risk.Transfer().MinAmount)
	require.Equal(t, types.TransferTypeIsolatedMarginLow, risk.Transfer().Type)
	require.Equal(t, "party1", risk.Transfer().Owner)
	require.Equal(t, num.UintZero(), risk.MarginLevels().OrderMargin)
}

func TestSwithToIsolatedMarginContinuous(t *testing.T) {
	e := getTestEngine(t, num.DecimalOne())
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
	e.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Return(num.NewUint(999), nil).AnyTimes()

	// margin factor too low - 0.01 * 1000 * 1 = 10 < 31 initial margin
	_, err := e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), []*types.Order{}, num.DecimalFromFloat(0.01), nil)
	require.Equal(t, "required position margin must be greater than initial margin", err.Error())

	// not enough in general account to cover
	// required position margin (600) + require order margin (0) - margin balance (10) > general account balance (500)
	_, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), []*types.Order{}, num.DecimalFromFloat(0.6), nil)
	require.Equal(t, "insufficient balance in general account to cover for required order margin", err.Error())

	// case1 - need to topup margin account only
	risk, err := e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), []*types.Order{}, num.DecimalFromFloat(0.5), nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(risk))
	require.Equal(t, num.NewUint(490), risk[0].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(490), risk[0].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeMarginLow, risk[0].Transfer().Type)
	require.Equal(t, "party1", risk[0].Transfer().Owner)
	require.Equal(t, num.NewUint(0), risk[0].MarginLevels().OrderMargin)

	// case2 we have also some orders
	orders := []*types.Order{
		{Side: types.SideBuy, Remaining: 1, Price: num.NewUint(1000), Status: types.OrderStatusActive},
	}

	// not enough in general account to cover
	// required position margin (300) + require order margin (300) - margin balance (10) > general account balance (500)
	_, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, num.DecimalFromFloat(0.3), nil)
	require.Equal(t, "insufficient balance in general account to cover for required order margin", err.Error())

	evt.general = 10000
	risk, err = e.SwitchToIsolatedMargin(context.Background(), evt, num.NewUint(100), num.DecimalOne(), orders, num.DecimalFromFloat(0.3), nil)
	require.NoError(t, err)
	require.Equal(t, 2, len(risk))
	require.Equal(t, num.NewUint(290), risk[0].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(290), risk[0].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeMarginLow, risk[0].Transfer().Type)
	require.Equal(t, "party1", risk[0].Transfer().Owner)
	require.Equal(t, num.NewUint(300), risk[0].MarginLevels().OrderMargin)

	require.Equal(t, num.NewUint(300), risk[1].Transfer().Amount.Amount)
	require.Equal(t, num.NewUint(300), risk[1].Transfer().MinAmount)
	require.Equal(t, types.TransferTypeOrderMarginLow, risk[1].Transfer().Type)
	require.Equal(t, "party1", risk[1].Transfer().Owner)
	require.Equal(t, num.NewUint(300), risk[0].MarginLevels().OrderMargin)
}
