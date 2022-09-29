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

package products_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestScalingOfSettlementPrice(t *testing.T) {
	t.Run("No scaling needed for settlement price for asset decimals", testNoScalingNeeded)
	t.Run("Need to scale up the settlement price for asset decimals", testScalingUpNeeded)
	t.Run("Need to scale down the settlement price for asset decimals no loss of precision", testScalingDownNeeded)
	t.Run("Need to scale down the settlement price for asset decimals with loss of precision", testScalingDownNeededWithPrecisionLoss)
}

func testNoScalingNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 5 decimal places => no scaling
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 5)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100000), scaled)
}

func testScalingUpNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 10 decimal places => x10^5
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 10)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(10000000000), scaled)
}

func testScalingDownNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 3)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1000), scaled)
}

func testScalingDownNeededWithPrecisionLoss(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(123456), 3)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1234), scaled)
}

type tstFuture struct {
	oe     *mocks.MockOracleEngine
	future *products.Future
}

func testFuture(t *testing.T) *tstFuture {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	f := &types.Future{
		SettlementAsset: "ETH",
		QuoteName:       "ETH",
		OracleSpecForSettlementData: &types.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*types.OracleSpecFilter{
				{
					Key: &types.OracleSpecPropertyKey{
						Name: "price.ETH.value",
						Type: oraclespb.PropertyKey_TYPE_INTEGER,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecForTradingTermination: &types.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*types.OracleSpecFilter{
				{
					Key: &types.OracleSpecPropertyKey{
						Name: "trading.termination",
						Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecBinding: &types.OracleSpecBindingForFuture{
			SettlementPriceProperty:    "price.ETH.value",
			TradingTerminationProperty: "trading.termination",
		},
		SettlementDataDecimals: 5,
	}

	ctx := context.Background()
	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(1), func(ctx context.Context, sid oracles.SubscriptionID) {})

	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(subscriptionID(2), func(ctx context.Context, sid oracles.SubscriptionID) {})

	future, err := products.NewFuture(ctx, log, f, oe)
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	return &tstFuture{
		future: future,
		oe:     oe,
	}
}

func subscriptionID(i uint64) oracles.SubscriptionID {
	return oracles.SubscriptionID(i)
}
